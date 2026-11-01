import unittest
from PIL import Image, ImageDraw
from notation import notation, flattened, plain_han_label, agreed_han_label
from regions import Components

class PolicyTests(unittest.TestCase):
    def test_chinese_caption_does_not_accept_flattened_units(self):
        self.assertTrue(plain_han_label('图1 霍尔效应原理图'))
        for text in ['表1 I-UAK关系', '光阑4mm', '电流10-11A', '十']:
            self.assertFalse(plain_han_label(text))

    def test_single_han_and_panel_captions(self):
        for text in ['红', '黑', '甲', '乙', '(a)封装外形图', '(b)内部示意图', '（A）剖面图', '(12)接线图']:
            self.assertTrue(agreed_han_label(text, .999, text, .999), text)
        self.assertTrue(agreed_han_label('(a)封装外形图', .95, '(a)封装外形图', .96))
        self.assertFalse(agreed_han_label('红', .97, '红', .999))
        self.assertFalse(agreed_han_label('红', .999, '纤', .999))
        for text in ['一', '二', '十', '丨', '(a)表1 I-UAK关系', '接4mm导线']:
            self.assertFalse(agreed_han_label(text, 1., text, 1.), text)

    def test_supported_notation(self):
        for value, expected in [(r'F_{\mathrm{E}}','F_{E}'), (r'E_H','E_{H}'), (r'A^{\prime}',"A'"), (r'10^{-11}','10^{-11}'), (r'\mathrm{\mathit{F}}_{B}','F_{B}')]:
            self.assertEqual(notation(value), expected)
        self.assertEqual(flattened('F_{E}'), 'FE')
        self.assertNotEqual(flattened('F_{E}'), flattened('Fε'))

    def test_unsupported_notation_is_not_flattened(self):
        for value in [r'\vec{E}',r'\frac{a}{b}',r'F_{a_{b}}',r'A_{2}^{3}',r'_H',r'\unknown{a}','']:
            self.assertIsNone(notation(value), value)

    def test_crossing_axis_excluded_from_label_crop(self):
        im=Image.new('RGB',(200,100),'white');d=ImageDraw.Draw(im)
        d.line((0,50,199,50),fill='black',width=2)
        d.rectangle((80,20,83,38),fill='black')
        d.rectangle((88,32,94,40),fill='black')
        box=Components(im).label_box([[76,16],[98,16],[98,54],[76,54]])
        self.assertEqual(box,[80,20,95,41])
        self.assertIsNone(Components(im).label_box([[20,46],[40,46],[40,55],[20,55]]))


class FractionPolicyTests(unittest.TestCase):
    def test_formula_formatting_and_fraction_structure(self):
        from fraction_policy import fraction_notation
        source = r'U_{\mathrm{\tiny~ H}}=R_{\mathrm{\tiny~ H}} \; \frac{I_{\mathrm{\tiny~ S}} B} {d}'
        self.assertEqual(fraction_notation(source), r'U_{H}=R_{H}\frac{I_{S}B}{d}')
        self.assertEqual(fraction_notation(r'q\frac{U_H}{h}=qvB'), r'q\frac{U_{H}}{h}=qvB')
        for value in [r'\frac{}{b}',r'\frac{a}{}',r'\frac{a}{b',r'\frac{a}{b}}',r'\frac{\sqrt{a}}{b}',r'\frac{a_{b_c}}{d}',r'\frac{a}{b}_c']:
            self.assertIsNone(fraction_notation(value),value)

    def test_real_formula_requires_geometry_and_ocr_agreement(self):
        from fraction_policy import fraction_candidate, agreed_fraction
        from pathlib import Path
        path=Path(__file__).parents[2]/'engine/testdata/hall-voltage-fraction.png'
        im=Image.open(path);labels=[dict(text='UH = R H',score=.952),dict(text='IB',score=.783)]
        self.assertEqual(fraction_candidate(Components(im),labels),[16,18,125,65])
        good=dict(text=r'U_H=R_H\frac{I_S B}{d}',score=.94)
        self.assertEqual(agreed_fraction(good,labels),r'U_{H}=R_{H}\frac{I_{S}B}{d}')
        self.assertIsNone(agreed_fraction(dict(good,score=.89),labels))
        self.assertIsNone(agreed_fraction(dict(good,text=r'V_H=R_H\frac{I_S B}{d}'),labels))
        self.assertIsNone(fraction_candidate(Components(im),[dict(text='电路图',score=1)]))
        plain=Image.new('RGB',(160,100),'white');d=ImageDraw.Draw(plain)
        d.text((10,40),'UH = RH',fill='black');d.line((90,40,140,40),fill='black')
        self.assertIsNone(fraction_candidate(Components(plain),labels))

if __name__=='__main__': unittest.main()
