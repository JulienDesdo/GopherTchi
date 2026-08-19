#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#include "smapp_darwin.h"
#include <stdlib.h>
#include <string.h>

static bool gophertchi_is_app_bundle(void) {
	@autoreleasepool {
		NSString *path = [[NSBundle mainBundle] bundlePath];
		return path != nil && [path hasSuffix:@".app"];
	}
}

bool gophertchi_login_supported(void) {
	if (@available(macOS 13.0, *)) {
		return gophertchi_is_app_bundle();
	}
	return false;
}

int gophertchi_login_status(void) {
	if (@available(macOS 13.0, *)) {
		if (!gophertchi_is_app_bundle()) {
			return GOPHER_LOGIN_UNSUPPORTED;
		}
		@autoreleasepool {
			SMAppServiceStatus st = [SMAppService mainAppService].status;
			switch (st) {
			case SMAppServiceStatusEnabled:
				return GOPHER_LOGIN_ENABLED;
			case SMAppServiceStatusRequiresApproval:
				return GOPHER_LOGIN_REQUIRES_APPROVAL;
			case SMAppServiceStatusNotRegistered:
				return GOPHER_LOGIN_NOT_REGISTERED;
			case SMAppServiceStatusNotFound:
				return GOPHER_LOGIN_NOT_FOUND;
			default:
				return GOPHER_LOGIN_ERROR;
			}
		}
	}
	return GOPHER_LOGIN_UNSUPPORTED;
}

bool gophertchi_login_set(bool enable, char **error_message) {
	if (error_message) {
		*error_message = NULL;
	}
	if (@available(macOS 13.0, *)) {
		if (!gophertchi_is_app_bundle()) {
			if (error_message) {
				*error_message = strdup("not running from a .app bundle");
			}
			return false;
		}
		@autoreleasepool {
			SMAppService *service = [SMAppService mainAppService];
			NSError *error = nil;
			BOOL ok = NO;
			if (enable) {
				ok = [service registerAndReturnError:&error];
			} else {
				ok = [service unregisterAndReturnError:&error];
			}
			if (!ok && error != nil && error_message) {
				const char *msg = error.localizedDescription.UTF8String;
				if (msg) {
					*error_message = strdup(msg);
				}
			}
			return ok ? true : false;
		}
	}
	if (error_message) {
		*error_message = strdup("requires macOS 13 or later");
	}
	return false;
}

void gophertchi_login_free(char *p) {
	free(p);
}
