// epoll 源码分析 linux kernel: v5.8.7
问题:
1. 往 epoll 中注册事件发生了什么? 被注册 fd 如何跟 epoll 交互?
猜测:

2. EPOLL*各个选项的作用?
由特定的 fd 的 poll 函数决定
3. 如何确定 fd 的事件类型?
通过 file_operations-> poll 函数确定,我们可以看看 tcp 套接字 poll 函数的实现, 函数路径: net/ipv4/tcp.c -> tcp_poll

4. 各类回调函数如何如何统一?
