# Minecraft Reverse Proxy

A server that redirects Minecraft connections to specified hosts based on the hostname.  
For example, connecting from **minecraft-a.example.com** could redirect you to one server, whereas **minecraft-b.example.com** would direct you to another, without requiring the servers to be setup on different IPs, or requiring SRV records to be changed.  

## Configuration
Configuration is done in the **config.json** file, in this format:
```json
{
  "Servers": {
    "hostname.example.com": "destination-server:25565"
  }
}
```
Destination server must always include the port, even if it is default.  
The reverse proxy listens on port 25565.

The config file will be reloaded whenever a new client connects, so adding a new server to the config doesn't require a restart.

## To Build
To build this project:
```shell script
git clone https://github.com/UnacceptableUse/minecraft-proxy
go build -o minecraft-proxy
./minecraft-proxy
```