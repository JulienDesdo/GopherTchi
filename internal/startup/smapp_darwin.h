#ifndef GOPHERTCHI_SMAPP_DARWIN_H
#define GOPHERTCHI_SMAPP_DARWIN_H

#include <stdbool.h>

enum {
	GOPHER_LOGIN_UNSUPPORTED = 0,
	GOPHER_LOGIN_ENABLED = 1,
	GOPHER_LOGIN_NOT_REGISTERED = 2,
	GOPHER_LOGIN_REQUIRES_APPROVAL = 3,
	GOPHER_LOGIN_NOT_FOUND = 4,
	GOPHER_LOGIN_ERROR = 5
};

bool gophertchi_login_supported(void);
int gophertchi_login_status(void);
bool gophertchi_login_set(bool enable, char **error_message);
void gophertchi_login_open_settings(void);
void gophertchi_login_free(char *p);

#endif
