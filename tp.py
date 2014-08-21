# -*- coding:utf-8 -*-
'''
epoll的例子
接受+转发的例子


epoll工作模式分为edge-trigger 和 level-trigger, 默认是 LT，设置方式分别是：
epoll.register(serversocket.fileno(), select.EPOLLIN | select.EPOLLET)  #ET
epoll.register(serversocket.fileno(), select.EPOLLIN )                                  #LT
两者的区别是，ET事件触发只会在读/写从unready到ready跃迁时发生， 而LT是只要ready就会触发。想象有个readylist， ET是事件触发一次之后就将fd从list里面去除掉，而LT不会，LT会等到fd为unready时才会从这个list去除掉。理论上ET更节省内存，但ET需要更复杂得逻辑和更好得控制，所以一般使用LT。

'''
import socket,select
import struct
import hashlib
import string

ip = "0.0.0.0"
port = int(9040)

def get_table(key):
    m = hashlib.md5()
    m.update(key)
    s = m.digest()
    (a, b) = struct.unpack('<QQ', s)
    table = [c for c in string.maketrans('', '')]
    for i in xrange(1, 1024):
        table.sort(lambda x, y: int(a % (ord(x) + i) - a % (ord(y) + i)))
    return table

def encrypt(data):
    return data.translate(encrypt_table)

def decrypt(data):
    return data.translate(decrypt_table)

rhost = ('106.186.100.100',1080)
password = 'password'
encrypt_table = ''.join(get_table(password))
decrypt_table = string.maketrans(encrypt_table, string.maketrans('', ''))


SO_ORIGINAL_DST = 80 #tcp的协议是80 linux/netfilter_ipv4.h:#define SO_ORIGINAL_DST 80

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR, 1)

s.setblocking(0) #设置不是堵塞
s.bind((ip,port))
s.listen(50) #允许最多5个连接

connections = {}

epoll = select.epoll() #创建一个epoll对象。
epoll.register(s.fileno(), select.EPOLLIN) #在服务端socket上面注册对读event的关注。一个读event随时会触发服务端socket去接收一个socket连接。

while True:
  events = epoll.poll(1) #查询epoll对象，看是否有任何关注的event被触发。参数“1”表示，我们会等待1秒来看是否有event发生。如果有任何我们感兴趣的event发生在这次查询之前，这个查询就会带着这些event的列表立即返回。

  for fileno,event in events: #event作为一个序列（fileno，event code）的元组返回。fileno是文件描述符的代名词，始终是一个整数。
    #如果是socket有新数据到来
    if fileno == s.fileno():
      print 'new client joined,accpeting..'
      conn,clientAddr = s.accept()

      #下面获取iptables 过后的目标地址和端口
      dst = conn.getsockopt(socket.SOL_IP,SO_ORIGINAL_DST,16)
      #关于struct,可以参考 http://www.cnblogs.com/gala/archive/2011/09/22/2184801.html x是跳过的意思
      _, dst_port, ip1, ip2, ip3, ip4 = struct.unpack("!HHBBBB8x", dst)
      dst_ip = '%s.%s.%s.%s' % (ip1,ip2,ip3,ip4)
      print "target is %s:%s" % (dst_ip,dst_port)

      remote = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

      addrtype = chr(1)
      dst_ip = socket.inet_aton(dst_ip)
      dst_port = struct.pack('>H', dst_port)
      data = addrtype + dst_ip + dst_port
      #remote.setblocking(0)
      try:
        remote.connect(rhost)
      except socket.error as e:
        print(str(e))
      else:
        print "We are connected to %s:%d" % remote.getpeername()
      remote.send(encrypt(data))
      #监视的对象列表，要加多一个conn的
      epoll.register(conn.fileno(),select.EPOLLIN)
      epoll.register(remote.fileno(),select.EPOLLIN)

      connections[conn.fileno()] = remote
      connections[remote.fileno()] = conn

    elif event & select.EPOLLIN: #如果event=1 那就是说有数据进入。select.EPOLLIN的值就是1.所有 1 & 1是true。
      data = connections[connections[fileno].fileno()].recv(1024)
      if '10.0.0.144' in connections[connections[fileno].fileno()].getpeername():
        print "本地数据"
        data = encrypt(data)
      else:
        print "服务器数据"
        data = decrypt(data)

      connections[fileno].send(data)
      if not data:
        #如果断开了，就把这个对象，从监视对象列表中删除掉。
        print "remove %d" % fileno
        print "remove %d" % connections[fileno].fileno()
        epoll.unregister(fileno)
        epoll.unregister(connections[fileno].fileno())

        connections[connections[fileno].fileno()].close()
        connections[fileno].close()

        #del connections[fileno]
        #del connections[connections[fileno].fileno()]
      else:
        pass
        print "data coming"

'''
TCP_CORK和TCP_NODELAY:
设置方式:
connections[fileno].setsockopt(socket.IPPROTO_TCP, socket.TCP_CORK, 1)
s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
理解这个两个参数要先理解Nagle算法。 Nagle算法被设计用来解决拥塞网络中大量小封包问题，方法就是在缓存数据包，在缓存区未满且时间未超过阈值时，不直接发送数据包，将多个数据包组合起来一起发送，提供网络使用率。TCP中默认是开启该算法，TCP_CORK和TCP_NODELAY被用来禁止Nagle算法，但他们是互斥得，使用环境和目的不一样。
TCP_NODELAY适用于要求实时数据发送环境，不希望有延时。比如ssh
TCP_CORK是自己控制数据包发送时机，原理是先“塞”TCP,然后往缓冲区写数据，然后打开塞子，将数据一起发出，适合发送大量数据，且小的封包也会被发送，不会被缓存住。比如http， ftp
'''
