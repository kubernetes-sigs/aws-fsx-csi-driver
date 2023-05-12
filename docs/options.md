# Driver Options
There are a couple of driver options that can be passed as arguments when starting the driver container.

| Option argument             | value sample                                      | default                                             | Description         |
|-----------------------------|---------------------------------------------------|-----------------------------------------------------|---------------------|
| endpoint                    | tcp://127.0.0.1:10000/                            | unix:///var/lib/csi/sockets/pluginproxy/csi.sock    | The socket on which the driver will listen for CSI RPCs|
| logging-format              | json                                              | text                                                | Sets the log format. Permitted formats: text, json|
