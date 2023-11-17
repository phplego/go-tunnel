#!/bin/bash

# Import service name and paths
source ./service-config.sh

# Stop the service
sudo systemctl stop ${SERVICE_NAME}

# Disable the service
sudo systemctl disable ${SERVICE_NAME}

# Remove the service file
sudo rm ${SERVICE_FILE}

# Reload the systemd daemon
sudo systemctl daemon-reload

# Print a message indicating that the uninstallation is complete
echo "The ${SERVICE_NAME} service has been uninstalled."
