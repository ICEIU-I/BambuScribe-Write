// Package project stores one editable handwriting document. The archive keeps
// the manifest readable and every raster source in its original bytes, so a
// saved file stays small and a later resize can still regenerate clean strokes.
package project

import (
	"archive/zip"
	"bambuscribe/desktop/internal/engine"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path"
	"strings"
)

const (
	// Extension is the suffix the desktop editor reads and writes.
	Extension = ".ccf"
	// Version is the format revision written into every manifest.
	Version = 6

	manifestName = "project.json"
	imageDir     = "images"
	maxManifest  = 4 << 20
	maxImageData = 12 << 20
)

// Project is the editable state of one document: source text, paper layout,
// device writing parameters and the raster images with their corrections.
// Generated strokes are not stored, because preview rebuilds them from these
// inputs and from the embedded handwriting font.
type Project struct {
	Erasures []engine.PageErasure   `json:"erasures,omitempty"`
	Version  int                    `json:"version"`
	Text     string                 `json:"text"`
	Layout   engine.LayoutSettings  `json:"layout"`
	Writing  engine.WritingSettings `json:"writing"`
	Images   []engine.ImageAsset    `json:"images"`
	Tables   []engine.TableAsset    `json:"tables,omitempty"`
}

// Encode writes the project as a zip archive. Each image leaves its base64
// data URL for its own stored entry, which keeps the manifest readable and
// avoids deflating PNG or JPEG payloads that are already compressed.
func Encode(p Project) ([]byte, error) {
	if err := engine.ValidateErasures(p.Erasures); err != nil {
		return nil, err
	}
	if err := engine.ValidateTables(p.Tables); err != nil {
		return nil, err
	}
	if err := p.Layout.Validate(); err != nil {
		return nil, err
	}
	stored := p
	stored.Version = Version
	stored.Images = make([]engine.ImageAsset, len(p.Images))
	seen := make(map[string]bool, len(p.Images))
	var b bytes.Buffer
	out := zip.NewWriter(&b)
	for i, asset := range p.Images {
		if asset.Diagram != nil {
			if err := asset.Diagram.Validate(); err != nil {
				return nil, err
			}
		}
		if !validImageID(asset.ID) {
			return nil, fmt.Errorf("图片标识不合法：%q", asset.ID)
		}
		if seen[asset.ID] {
			return nil, fmt.Errorf("图片标识重复：%s", asset.ID)
		}
		seen[asset.ID] = true
		raw, format, err := imageBytes(asset.Data)
		if err != nil {
			return nil, err
		}
		if err = writeEntry(out, path.Join(imageDir, asset.ID+"."+format), raw, zip.Store); err != nil {
			return nil, err
		}
		stored.Images[i] = asset
		stored.Images[i].Data = ""
	}
	body, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	if len(body) > maxManifest {
		return nil, fmt.Errorf("项目内容过多，请拆分后保存")
	}
	if err = writeEntry(out, manifestName, body, zip.Deflate); err != nil {
		return nil, err
	}
	if err = out.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Decode rebuilds a project from an archive. A newer format revision is
// rejected rather than silently dropping the fields this build cannot read.
func Decode(data []byte) (Project, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Project{}, fmt.Errorf("项目文件已损坏：%w", err)
	}
	var manifest *zip.File
	images := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		name := strings.TrimPrefix(file.Name, "./")
		if name == manifestName {
			manifest = file
			continue
		}
		if dir, base := path.Split(name); dir == imageDir+"/" && base != "" {
			images[strings.TrimSuffix(base, path.Ext(base))] = file
		}
	}
	if manifest == nil {
		return Project{}, fmt.Errorf("项目文件缺少 %s", manifestName)
	}
	body, err := readEntry(manifest, maxManifest)
	if err != nil {
		return Project{}, err
	}
	var p Project
	if err = json.Unmarshal(body, &p); err != nil {
		return Project{}, fmt.Errorf("项目文件无法解析：%w", err)
	}
	if p.Version < 1 || p.Version > Version {
		return Project{}, fmt.Errorf("不支持的项目版本 %d，请升级应用", p.Version)
	}
	if err = p.Layout.Validate(); err != nil {
		return Project{}, err
	}
	if err = engine.ValidateTables(p.Tables); err != nil {
		return Project{}, err
	}
	if err = engine.ValidateErasures(p.Erasures); err != nil {
		return Project{}, err
	}
	for i, asset := range p.Images {
		if asset.Diagram != nil {
			if err := asset.Diagram.Validate(); err != nil {
				return Project{}, err
			}
		}
		file := images[asset.ID]
		if file == nil {
			return Project{}, fmt.Errorf("项目缺少图片 %s", asset.ID)
		}
		raw, err := readEntry(file, maxImageData)
		if err != nil {
			return Project{}, err
		}
		if p.Images[i].Data, err = dataURL(raw); err != nil {
			return Project{}, err
		}
	}
	return p, nil
}

// dataURL restores the base64 form the engine expects, taking the media type
// from the bytes themselves instead of trusting the archived entry name.
func dataURL(raw []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("仅支持 PNG 或 JPEG 图片")
	}
	return "data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func imageBytes(data string) ([]byte, string, error) {
	if len(data) == 0 || len(data) > maxImageData {
		return nil, "", fmt.Errorf("图片大小须不超过 12 MB")
	}
	comma := strings.IndexByte(data, ',')
	if comma < 0 || !strings.Contains(strings.ToLower(data[:comma]), "base64") {
		return nil, "", fmt.Errorf("图片数据格式不正确")
	}
	raw, err := base64.StdEncoding.DecodeString(data[comma+1:])
	if err != nil {
		return nil, "", fmt.Errorf("图片无法读取：%w", err)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, "", fmt.Errorf("仅支持 PNG 或 JPEG 图片")
	}
	return raw, format, nil
}

// validImageID keeps an identifier usable as a single archive entry name.
func validImageID(id string) bool {
	return id != "" && !strings.HasPrefix(id, ".") && !strings.ContainsAny(id, `/\`)
}

func writeEntry(out *zip.Writer, name string, body []byte, method uint16) error {
	w, err := out.CreateHeader(&zip.FileHeader{Name: name, Method: method})
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func readEntry(file *zip.File, limit int) ([]byte, error) {
	r, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("项目文件已损坏：%w", err)
	}
	defer r.Close()
	body, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("项目文件已损坏：%w", err)
	}
	if len(body) > limit {
		return nil, fmt.Errorf("%s 内容过大", file.Name)
	}
	return body, nil
}
