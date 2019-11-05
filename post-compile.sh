sudo cp /buildroot/bin/amb-sidecar /ambassador/sidecars
sudo touch /ambassador/.edge_stack
sudo mkdir -p /ambassador/webui/bindata && sudo rsync -a --delete /buildroot/apro/cmd/amb-sidecar/webui/bindata/  /ambassador/webui/bindata
