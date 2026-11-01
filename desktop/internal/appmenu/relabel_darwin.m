// Relabels the standard menus Wails builds from roles. Those items carry
// hardcoded English titles, but the selectors they drive are the standard
// AppKit ones, so the app can rename them without losing the responder-chain
// behaviour (cut/copy/paste, hide, quit, minimize) that only roles provide.
#import <Cocoa/Cocoa.h>
#include <stdlib.h>

static void Relabel(NSMenu *menu, NSDictionary *labels) {
    for (NSMenuItem *item in menu.itemArray) {
        if (item.submenu != nil) {
            NSString *title = labels[item.title] ?: labels[item.submenu.title];
            if (title != nil) {
                item.title = title;
                item.submenu.title = title;
            }
            Relabel(item.submenu, labels);
            continue;
        }
        if (item.action == nil) continue;
        // Wails' Window role uses an NSView selector without its required
        // parameters. Route the shortcut through the window's native toggle.
        if (item.action == @selector(enterFullScreenMode:)) {
            item.action = @selector(toggleFullScreen:);
        }
        NSString *key = NSStringFromSelector(item.action);
        if ([key hasSuffix:@":"]) key = [key substringToIndex:key.length - 1];
        NSString *title = labels[key];
        if (title != nil) item.title = title;
    }
}

void RelabelRoleMenus(const char *json) {
    @autoreleasepool {
        NSString *text = [NSString stringWithUTF8String:json];
        if (text == nil) return;
        NSData *data = [text dataUsingEncoding:NSUTF8StringEncoding];
        NSDictionary *labels = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
        if (![labels isKindOfClass:[NSDictionary class]]) return;
        // AppKit refuses main-menu edits from any thread but the main one,
        // and the menu was just rebuilt from a Go call.
        dispatch_async(dispatch_get_main_queue(), ^{
            NSMenu *main = [NSApplication sharedApplication].mainMenu;
            if (main == nil) return;
            Relabel(main, labels);
        });
    }
}
