package main

/*
#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <poll.h>
#include <scsi/scsi.h>

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

int tcmu_wait_for_next_command(struct tcmu_device *dev) {
	struct pollfd pfd;

	pfd.fd = tcmu_get_dev_fd(dev);
	pfd.events = POLLIN;
	pfd.revents = 0;

	poll(&pfd, 1, -1);

	if (pfd.revents != POLLIN) {
		errp("poll received unexpected revent: 0x%x\n", pfd.revents);
		return -1;
	}
	return 0;
}

uint8_t tcmucmd_get_scsi_cmd(struct tcmulib_cmd *cmd) {
	return cmd->cdb[0];
}

int tcmucmd_emulate_inquiry(struct tcmulib_cmd *cmd, struct tcmu_device *dev) {
	return tcmu_emulate_inquiry(dev,
			cmd->cdb, cmd->iovec, cmd->iov_cnt, cmd->sense_buf);
}

int tcmucmd_emulate_test_unit_ready(struct tcmulib_cmd *cmd) {
	return tcmu_emulate_test_unit_ready(cmd->cdb, cmd->iovec,
			cmd->iov_cnt, cmd->sense_buf);
}

int tcmucmd_emulate_service_action_in(struct tcmulib_cmd *cmd,
		uint64_t num_lbas, uint32_t block_size) {
	if (cmd->cdb[1] == READ_CAPACITY_16) {
		return tcmu_emulate_read_capacity_16(num_lbas,
			block_size,
			cmd->cdb, cmd->iovec, cmd->iov_cnt, cmd->sense_buf);
	}
	return TCMU_NOT_HANDLED;
}

int tcmucmd_emulate_mode_sense(struct tcmulib_cmd *cmd) {
	return tcmu_emulate_mode_sense(cmd->cdb, cmd->iovec, cmd->iov_cnt, cmd->sense_buf);
}

int tcmucmd_emulate_mode_select(struct tcmulib_cmd *cmd) {
	return tcmu_emulate_mode_select(cmd->cdb, cmd->iovec, cmd->iov_cnt, cmd->sense_buf);
}

int tcmucmd_set_medium_error(struct tcmulib_cmd *cmd) {
	return tcmu_set_sense_data(cmd->sense_buf, MEDIUM_ERROR, ASC_READ_ERROR, NULL);
}

uint64_t tcmucmd_get_lba(struct tcmulib_cmd *cmd) {
	return tcmu_get_lba(cmd->cdb);
}

uint32_t tcmucmd_get_xfer_length(struct tcmulib_cmd *cmd) {
	return tcmu_get_xfer_length(cmd->cdb);
}

void *allocate_buffer(int length) {
	return calloc(1, length);
}

int tcmucmd_memcpy_into_iovec(struct tcmulib_cmd *cmd, void *buf, int length) {
	return tcmu_memcpy_into_iovec(cmd->iovec, cmd->iov_cnt, buf, length);
}

int tcmucmd_memcpy_from_iovec(struct tcmulib_cmd *cmd, void *buf, int length) {
	return tcmu_memcpy_from_iovec(buf, length, cmd->iovec, cmd->iov_cnt);
}

*/
import "C"
