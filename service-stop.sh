#!/bin/bash

# Import service name and paths
source ./service-config.sh

# Stop the service
sudo systemctl stop ${SERVICE_NAME}

# Print a message indicating that the stopping is complete
echo "The ${SERVICE_NAME} service has been stopped."
