#!/bin/bash

# Import service name and paths
source ./service-config.sh

# Show logs
journalctl -u $SERVICE_NAME -f -n ${1:-1000} --output cat
