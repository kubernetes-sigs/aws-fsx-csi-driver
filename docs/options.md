# Driver Options
There are a couple of driver options that can be passed as arguments when starting the driver container.

| Option argument     | value sample            | default                                          | Description                                                                                      |
|---------------------|-------------------------|--------------------------------------------------|--------------------------------------------------------------------------------------------------|
| endpoint            | tcp://127.0.0.1:10000/  | unix:///var/lib/csi/sockets/pluginproxy/csi.sock | The socket on which the driver will listen for CSI RPCs                                          |
| extra-tags          | key1=value1,key2=value2 |                                                  | Tags specified in the controller spec are attached to each dynamically provisioned resource      |
| logging-format      | json                    | text                                             | Sets the log format. Permitted formats: text, json                                               |
| retry-taint-removal | true                    | false                                            | If set to true, will keep retrying node taint removal for 90 seconds with an exponential backoff |
