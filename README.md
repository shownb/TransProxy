TransProxy
==========

添加了一个socks5转透明代理的代码。来自 https://github.com/xsm1997/KumaSocks
```bash
sudo iptables -t nat -A PREROUTING -i eth1 -p tcp --syn -j REDIRECT --to-ports 9040
sudo sysctl -w net.ipv4.ip_forward=1
```
/etc/resolv.conf
```
options use-vc
nameserver 1.1.1.1
nameserver 8.8.8.8
```


TransProxy for shadowsocks.

自制，常有错

socks5.txt介绍socks5代理的协议。

用作透明代理。不再需要设置客户端。

例如，在自己路由设置一个vpn server，在iptables以下东西。

```
sudo iptables -t nat -A PREROUTING -i ppp0 -p udp --dport 53 -j REDIRECT --to-ports 5353
sudo iptables -t nat -A PREROUTING -i ppp0 -p tcp --syn -j REDIRECT --to-ports 9040
```

map[string]string 

for example:server extip 110.110.110.110 localip 10.0.0.1
client A may be extip 119.119.119.118 localip 10.0.0.2
client B may be extip 119.119.119.119 localip 10.0.0.3

server listen at 3306

when client A connect to server,create clients["119.119.119.118"]= "10.0.0.2"
and client B connect to server,create clients["119.119.119.119"]= "10.0.0.3"

server recive iface packet,open it and find the dst ip(for example 10.0.0.3) und send the udp send(119.119.119.119)
when server recive udp packet,just write to iface.

client recvie udp packet,just write to iface,and client revie iface packet,just send(110.110.110.110).
