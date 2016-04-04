# Longhorn

# Environment

1. Copy `controller/libs/libtcmu.so*` to your `/usr/lib`, in order to make TCMU work.
2. `make`

# Run

1. Start `replica` first. Run it in background as `./replica &`
2. Then use [`enable_tcmu.sh`](https://gist.github.com/yasker/866979552ad6aae581cc#file-enable_tcmu-sh) script to create a device with size 1073741824.
3. Start `controller` to connect to TCMU. You should have a new SCSI device now.

No parameter needed for now. Currect `controller` would look for the `replica` at `localhost:5000`.

