#!/bin/sh

cmdArgs="$*"
if [ -n "$cmdArgs" ]; then
  /opt/dnspod-http-dns $cmdArgs
  exit 0
fi

Args=${Args:--T -U --fallbackedns 119.29.29.29}

cat > /opt/supervisord.conf <<EOF
[supervisord]
nodaemon=true

[program:dnspod-http-dns]
command=/opt/dnspod-http-dns ${Args}
autorestart=true
redirect_stderr=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0

EOF

/usr/bin/supervisord -c /opt/supervisord.conf
