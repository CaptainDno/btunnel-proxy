Socks adapter for [btunnel](https://github.com/CaptainDno/btunnel).

## State of the project

Currently, only `CONNECT` Socks5 request is supported. Connection is not reestablished automatically on failure.
So this software should be barely enough to serf the web or do other similar staff.

## Usage

Example assumes you compiled project into executable called `bt`. (e.g. by calling `go build -o ./bin/bt ./cmd` from project root)

On your server execute:
```shell
./bt server keygen -n 10 example # Generate 10 keys for client with id "example"

# Now copy example.pgb to the client machine (using scp or any other method)

./bt server start 127.0.0.1:8080 # Replace IP and port with actual server IP (or 0.0.0.0) and one of ports used for Bittorrent
```

On your client execute:
```shell
./bt client connect --id test --listen 127.0.0.1:9999 127.0.0.1:8080 # Again, replace 127.0.0.1:8080 with server IP:PORT
```

Then you can verify that everything is working correctly:
```shell
curl --socks5 127.0.0.1:9999 https://www.google.com
```

## Speed

Well, it depends on the network environment. During my testing between Samara and Frankfurt, I got approximately 30 Megabit/second and very unstable ping. 