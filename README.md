TransProxy
==========

TransProxy for shadowsocks.

自制，常有错

socks5.txt介绍socks5代理的协议。

用作透明代理。不再需要设置客户端。

例如，在自己路由设置一个vpn server，在iptables以下东西。

```
sudo iptables -t nat -A PREROUTING -i ppp0 -p udp --dport 53 -j REDIRECT --to-ports 5353
sudo iptables -t nat -A PREROUTING -i ppp0 -p tcp --syn -j REDIRECT --to-ports 9040
```
