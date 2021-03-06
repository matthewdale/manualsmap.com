#!/bin/sh

# Use 'exec' to make './api' PID 1, replacing 'sh'
# and correctly forwarding signals.
exec ./api \
    --addr=:80 \
    --apple-team-id=$APPLE_TEAM_ID \
    --mapkit-key-id=$MAPKIT_KEY_ID \
    --mapkit-secret-b64=$MAPKIT_SECRET_B64 \
    --psql-conn=$PSQL_CONN \
    --recaptcha-secret=$RECAPTCHA_SECRET \
    --license-salt=$LICENSE_SALT \
    --cloudinary-secret=$CLOUDINARY_SECRET
