#!/bin/bash

# Import service name and paths
source ./service-config.sh

# Restart the service
sudo systemctl restart ${SERVICE_NAME}

# Print a message indicating that the restarting is complete
echo "The ${SERVICE_NAME} service has been restarted."
