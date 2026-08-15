//go:build darwin

#import <AppKit/AppKit.h>
#import <objc/runtime.h>
#import <string.h>

extern void molebarMenuTrackingEnded(void);

static NSInteger molebar_track_depth;
static NSTimeInterval molebar_last_stay_open;
static NSMenu *molebar_root_menu;
static void (*original_cancel_tracking)(id, SEL);
static void (*original_cancel_tracking_noanim)(id, SEL);

static BOOL molebar_keeps_menu_open(NSString *title) {
	static NSSet<NSString *> *titles;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		titles = [NSSet setWithArray:@[
			@"Minimal",
			@"Developer",
			@"Network",
			@"Battery",
			@"Full",
			@"CPU",
			@"Memory",
			@"RX",
			@"TX",
			@"Health",
			@"Temperature",
			@"Disk",
			@"Alerts",
			@"Launch MoleBar at Login",
		]];
	});
	return title != nil && [titles containsObject:title];
}

static void molebar_note_stay_open(void) {
	molebar_last_stay_open = [NSDate timeIntervalSinceReferenceDate];
}

static BOOL molebar_should_suppress_cancel(void) {
	return ([NSDate timeIntervalSinceReferenceDate] - molebar_last_stay_open) < 0.5;
}

static BOOL molebar_looks_like_ours(NSMenu *menu) {
	if (menu == nil) {
		return NO;
	}
	for (NSMenuItem *item in menu.itemArray) {
		if ([item.title isEqualToString:@"Tray Metrics"] || [item.title isEqualToString:@"Quit"] ||
		    molebar_keeps_menu_open(item.title)) {
			return YES;
		}
		if (item.hasSubmenu && molebar_looks_like_ours(item.submenu)) {
			return YES;
		}
	}
	return NO;
}

@interface MolebarToggleController : NSObject
- (void)clicked:(NSButton *)sender;
@end

@implementation MolebarToggleController
- (void)clicked:(NSButton *)sender {
	molebar_note_stay_open();
	NSMenuItem *item = sender.enclosingMenuItem;
	if (item == nil || item.action == NULL || item.target == nil) {
		return;
	}
	[NSApp sendAction:item.action to:item.target from:item];
}
@end

static MolebarToggleController *molebar_toggle_controller(void) {
	static MolebarToggleController *controller;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		controller = [[MolebarToggleController alloc] init];
	});
	return controller;
}

static void molebar_sync_button(NSMenuItem *item) {
	if (![item.view isKindOfClass:[NSButton class]]) {
		return;
	}
	NSButton *button = (NSButton *)item.view;
	button.title = item.title;
	button.state = item.state;
	button.enabled = item.enabled;
}

static void molebar_attach_checkbox(NSMenuItem *item, NSMenu *menu) {
	CGFloat width = menu.size.width;
	if (width < 200) {
		width = 240;
	}
	NSButton *button = [[NSButton alloc] initWithFrame:NSMakeRect(0, 0, width, 20)];
	button.buttonType = NSButtonTypeSwitch;
	button.title = item.title;
	button.state = item.state;
	button.enabled = item.enabled;
	button.bordered = NO;
	button.font = [NSFont menuFontOfSize:0];
	button.focusRingType = NSFocusRingTypeNone;
	button.ignoresMultiClick = YES;
	button.target = molebar_toggle_controller();
	button.action = @selector(clicked:);
	if (@available(macOS 10.14, *)) {
		button.appearance = menu.appearance ?: NSApp.effectiveAppearance;
	}
	item.view = button;
}

static void molebar_decorate_menu(NSMenu *menu) {
	if (menu == nil) {
		return;
	}
	for (NSMenuItem *item in menu.itemArray) {
		if (item.hasSubmenu) {
			molebar_decorate_menu(item.submenu);
			continue;
		}
		if (item.isSeparatorItem || !molebar_keeps_menu_open(item.title)) {
			continue;
		}
		if (item.view != nil) {
			molebar_sync_button(item);
			continue;
		}
		molebar_attach_checkbox(item, menu);
	}
}

static NSMenu *molebar_find_status_menu(void) {
	id delegate = [NSApp delegate];
	if (delegate == nil) {
		return nil;
	}
	unsigned int count = 0;
	Ivar *ivars = class_copyIvarList([delegate class], &count);
	NSMenu *found = nil;
	for (unsigned int i = 0; i < count && found == nil; i++) {
		const char *type = ivar_getTypeEncoding(ivars[i]);
		if (type == NULL) {
			continue;
		}
		id value = object_getIvar(delegate, ivars[i]);
		if (strstr(type, "NSStatusItem") != NULL && [value isKindOfClass:[NSStatusItem class]]) {
			NSMenu *menu = [(NSStatusItem *)value menu];
			if (molebar_looks_like_ours(menu)) {
				found = menu;
			}
		} else if (strstr(type, "NSMenu") != NULL && [value isKindOfClass:[NSMenu class]]) {
			if (molebar_looks_like_ours((NSMenu *)value)) {
				found = (NSMenu *)value;
			}
		}
	}
	free(ivars);
	return found;
}

static void molebar_cancel_tracking(id self, SEL sel) {
	if (molebar_should_suppress_cancel()) {
		return;
	}
	original_cancel_tracking(self, sel);
}

static void molebar_cancel_tracking_noanim(id self, SEL sel) {
	if (molebar_should_suppress_cancel()) {
		return;
	}
	original_cancel_tracking_noanim(self, sel);
}

static void molebar_swizzle(Class cls, SEL sel, IMP replacement, void **original) {
	Method method = class_getInstanceMethod(cls, sel);
	if (method == NULL) {
		return;
	}
	*original = (void *)method_getImplementation(method);
	method_setImplementation(method, replacement);
}

static void molebar_on_did_begin(NSNotification *note) {
	molebar_track_depth++;
	NSMenu *menu = note.object;
	if (![menu isKindOfClass:[NSMenu class]]) {
		return;
	}
	NSMenu *root = menu;
	while (root.supermenu != nil) {
		root = root.supermenu;
	}
	if (molebar_looks_like_ours(root)) {
		molebar_root_menu = root;
		molebar_decorate_menu(root);
	}
}

static void molebar_on_did_end(NSNotification *note) {
	(void)note;
	if (molebar_track_depth > 0) {
		molebar_track_depth--;
	}
	if (molebar_track_depth == 0) {
		molebarMenuTrackingEnded();
	}
}

int molebar_menu_is_tracking(void) {
	return molebar_track_depth > 0 ? 1 : 0;
}

void molebar_sync_stay_open_menu(void) {
	void (^sync)(void) = ^{
		if (molebar_root_menu == nil) {
			molebar_root_menu = molebar_find_status_menu();
		}
		if (molebar_root_menu != nil) {
			molebar_decorate_menu(molebar_root_menu);
		}
	};
	if ([NSThread isMainThread]) {
		sync();
		return;
	}
	dispatch_async(dispatch_get_main_queue(), sync);
}

void install_stay_open_menus(void) {
	void (^install)(void) = ^{
		static dispatch_once_t once;
		dispatch_once(&once, ^{
			Class menuClass = [NSMenu class];
			molebar_swizzle(menuClass, @selector(cancelTracking),
			                (IMP)molebar_cancel_tracking, (void **)&original_cancel_tracking);
			molebar_swizzle(menuClass, @selector(cancelTrackingWithoutAnimation),
			                (IMP)molebar_cancel_tracking_noanim, (void **)&original_cancel_tracking_noanim);

			NSNotificationCenter *center = [NSNotificationCenter defaultCenter];
			[center addObserverForName:NSMenuDidBeginTrackingNotification
			                    object:nil
			                     queue:nil
			                usingBlock:^(NSNotification *note) { molebar_on_did_begin(note); }];
			[center addObserverForName:NSMenuDidEndTrackingNotification
			                    object:nil
			                     queue:nil
			                usingBlock:^(NSNotification *note) { molebar_on_did_end(note); }];
		});
		molebar_root_menu = molebar_find_status_menu();
		if (molebar_root_menu != nil) {
			molebar_decorate_menu(molebar_root_menu);
		}
	};
	if ([NSThread isMainThread]) {
		install();
		return;
	}
	dispatch_sync(dispatch_get_main_queue(), install);
}
