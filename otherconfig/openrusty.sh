---------------------------------------------
# local ubuntu openresty config int  /usr/local/openresty/nginx/conf/nginx.conf
# Change user to www-data or your system user
user www-data;
worker_processes auto;

events {
    worker_connections 1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;

    # Basic settings
    sendfile        on;
    keepalive_timeout  65;

    # Main log format (without port)
    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    # Location-specific log format (with port)
    log_format  location_log  '$remote_addr - $remote_user [$time_local] "$request" '
                             '$status $body_bytes_sent "$http_referer" '
                             '"$http_user_agent" "$http_x_forwarded_for" '
                             'Port: $port';

    # Main access and error logs
    access_log  /var/log/nginx/access.log  main;
    error_log   /var/log/nginx/error.log;

    server {
        listen 8083;
        server_name ~^(?<port>\d+)\.code\.iamanshik\.online$;

        # Remove the debug log from server block
        # access_log /var/log/nginx/debug.log main;
        # error_log  /var/log/nginx/debug_error.log;

        location / {
            # Location-specific logging with port
            access_log /var/log/nginx/location_access.log location_log;
            error_log  /var/log/nginx/location_error.log;

            # Log the proxy_pass URL
            add_header X-Debug-Proxy-Pass "http://127.0.0.1:$port";
            
            # Fix proxy_pass to use correct variable
            proxy_pass http://127.0.0.1:$port;
            
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # WebSocket support
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}








# worker_processes 1;
# events {
#     worker_connections 1024;
# }
# http {
#     server {
#         listen 8083 reuseport;
#         location / {
#             default_type text/plain;
#             content_by_lua_block {
#                 ngx.say("Hello World")
#             }
#         }
#     }
# }
------------------------------------------------------------
# aws ubuntu config nginx
server {
    listen 80;
    server_name iamanshik.online;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

    }
}

server {
    listen 80;
    server_name api.iamanshik.online;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

    }
}

server {
    listen 80;
    server_name *.code.iamanshik.online;

    location / {
        proxy_pass http://localhost:8083;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

    }
}

