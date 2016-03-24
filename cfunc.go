package main

/*
#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <poll.h>

#include "libtcmu.h"

void errp(const char *fmt, ...)
{
	va_list va;

	va_start(va, fmt);
	vfprintf(stderr, fmt, va);
	va_end(va);
}

int sh_open_cgo(struct tcmu_device *dev) {
	errp("open called\n");
	return shOpen(dev);
}

void sh_close_cgo(struct tcmu_device *dev) {
	shClose(dev);
}

static struct tcmulib_handler sh_handler = {
	.name = "Shorthorn TCMU handler",
	.subtype = "file",
	.cfg_desc = "dev_config=file/<path>",
	.added = sh_open_cgo,
	.removed = sh_close_cgo,
};

struct tcmulib_context *tcmu_init() {
	return tcmulib_initialize(&sh_handler, 1, errp);
}

bool tcmu_poll_master_fd(struct tcmulib_context *cxt) {
	int ret;
	struct pollfd pfd;

	pfd.fd = tcmulib_get_master_fd(cxt);
	pfd.events = POLLIN;
	pfd.revents = 0;

	ret = poll(&pfd, 1, -1);
	if (ret < 0) {
		errp("poll error out with %d", ret);
		exit(1);
	}

	if (pfd.revents) {
		tcmulib_master_fd_ready(cxt);
		return true;
	}
	return false;
}

*/
import "C"
