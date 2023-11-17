#!/bin/bash

export SERVICE_NAME="go-tunnel"
export SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
export EXECUTABLE_PATH="$(pwd)/${SERVICE_NAME}"