#import <Cocoa/Cocoa.h>

void SetWindowTheme(int dark) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSString *appearanceName = NSAppearanceNameAqua;
        if (dark) {
            if (@available(macOS 10.14, *)) appearanceName = NSAppearanceNameDarkAqua;
            else appearanceName = NSAppearanceNameVibrantDark;
        }
        NSAppearance *appearance = [NSAppearance appearanceNamed:appearanceName];
        CGFloat channel = dark ? 24.0 / 255.0 : 1.0;
        NSColor *background = [NSColor colorWithSRGBRed:channel green:channel blue:channel alpha:1.0];
        // Scope this to editor windows; native open/save panels keep their
        // standard AppKit appearance and responder behaviour.
        Class editorWindow = NSClassFromString(@"WailsWindow");
        for (NSWindow *window in NSApp.windows) {
            if (![window isKindOfClass:editorWindow]) continue;
            window.appearance = appearance;
            window.backgroundColor = background;
            window.titlebarAppearsTransparent = YES;
        }
    });
}
