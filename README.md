# H2c Client
![travis](https://travis-ci.org/YuShuanHsieh/h2c-client.svg?branch=master)

A simple CLI tool for testing h2c server. The tool use `HTTP/1.1 upgrade mechanism` to ask HTTP/2 support and try to establish a HTTP/2 connection with server upon receiving 101 switching protocols response. Please make sure your test server has HTTP/1.1 upgrade feature.

## Commands

### 1. connect
Connect to a given host server.

`connect <host>` 

Example:

`connect localhost:9090`

### 2. settings
Change settings values.

#### Supported settings ([RFC 7540 6.5.2](https://tools.ietf.org/html/rfc7540#section-6.5.2))
| Setting | Description  | Default | 
|---|---|---|
| push | This setting can be used to enable server push  | 0  | 
|  maxStream | Indicates the maximum number of concurrent streams that the sender will allow.  | 250 |
| windowSize  | Indicates the sender's initial window size for stream-level flow control |  65535 |
| frameSize  | Indicates the size of the largest frame payload that the sender is willing to receive |  16384 |

Example:

`settings maxStream=300 frameSize=5000`

#### Note
The settings will be sent to server after updating if it is connected state.

### 3. send

Send a request to a server.

#### Support Option
| Setting | Description  |
|---|---|---|
| PING | send PING to a server  |
|  GET <path> | send get request to a server  |

Example:

`send PING`  
`send GET /info`

### 4. close

Close current connection

Example:

`close`

### 5. exit

leave the client program. 

Example:

`exit`
