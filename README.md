TransProxy
==========

添加了一个socks5转透明代理的代码。来自 https://github.com/xsm1997/KumaSocks
```bash
# 可选：把规则放到自定义链，便于管理
sudo iptables -t nat -N TRANSPROXY

# 1) 从 PREROUTING 仅把经 eth0 的报文导入自定义链
sudo iptables -t nat -A PREROUTING -i eth0 -p tcp -j TRANSPROXY

# 2) 自定义链内：排除发往本机的目的地址
sudo iptables -t nat -A TRANSPROXY -m addrtype --dst-type LOCAL -j RETURN

# 3) 排除已经是 9040 的流量，避免自吃
sudo iptables -t nat -A TRANSPROXY -p tcp --dport 9040 -j RETURN

# 4) （可选）排除你不想代理的目的端口，比如 22
#sudo iptables -t nat -A TRANSPROXY -p tcp --dport 22 -j RETURN

# 5) 其余新建连接统一重定向到 9040
sudo iptables -t nat -A TRANSPROXY -p tcp -m conntrack --ctstate NEW -j REDIRECT --to-ports 9040
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
