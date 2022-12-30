/*
    使用 epoll 实现一个简单的 echo server
*/

#include <fcntl.h>
#include <unistd.h>
#include <sys/epoll.h>
#include <stdio.h>
#include <errno.h>
#include <malloc.h>
#include <string.h>
#include <stddef.h>
#include <signal.h>
#include <stdlib.h>
// #define DEBUG 1

// void debug_printf(const char* str, ...){
//     if(DEBUG != 1) {return;}
//     printf(str, ...);
// }

#define EPOLL_SIZE 1024
#define EPOLL_RECIVED_EVENTS 1024

void finalizer(int fd, int epfd, void* to_free){
    if(fd != STDIN_FILENO){close(fd);}
    close(epfd);
    free(to_free);
}

#define BUFSIZE 1
// 读取 fd 所有数据并输出到标准输出
void read_all_data(int fd) {
    char buf[BUFSIZE];
    for(int i = 0; i < 1; i++){
        int readn = read(fd, buf, BUFSIZE);
        if(readn == -1){
            if(errno == EINTR){
                continue;
            }
            if(errno == EAGAIN || errno == EWOULDBLOCK){
                break;
            }
            perror("readn error");
            return;
        }
        // printf("%d\n", readn);
        write(STDOUT_FILENO, buf, readn);
        if(readn < BUFSIZE) {
            break;
        }
    }
    write(STDOUT_FILENO, "\n", 1);
}

void handle_events(int epfd, struct epoll_event* events, size_t len) {
    for(int i = 0; i < len; i++) {
        struct epoll_event cur_event = events[i];
        int cur_fd = cur_event.data.fd;

        // 数据可读
        if(cur_event.events & EPOLLIN) {
            read_all_data(cur_fd);
        }
    }
}

void set_fd_nonblock(int fd){
    int attr = fcntl(fd, F_GETFL);
    attr |= O_NONBLOCK;
    fcntl(fd, F_SETFL, attr);
}

void boot_server() {
    // int fd = open("./file.txt", O_RDONLY|O_CREAT|O_NONBLOCK, 0666);
    // if(fd == -1) {
    //     perror("open ./file.txt error");
    //     return;
    // }
    set_fd_nonblock(STDIN_FILENO);

    int epfd = epoll_create(EPOLL_SIZE);
    if(epfd == -1){
        perror("create epoll instace error");
        // close(fd);
        return;
    }

    struct epoll_event* event = (struct epoll_event*) malloc(sizeof(struct epoll_event));
    memset(event, 0, sizeof(struct epoll_event));

    event->events = EPOLLIN | EPOLLERR | EPOLLET;
    event->data.fd = STDIN_FILENO;

    int ret = epoll_ctl(epfd, EPOLL_CTL_ADD, event->data.fd, event);
    if(ret != 0) {
        perror("epoll_ctl error");
        finalizer(event->data.fd, epfd, event);
        return;
    }

    struct epoll_event recived_events[EPOLL_RECIVED_EVENTS];
    for(;;){
        ret = epoll_wait(epfd, recived_events, EPOLL_RECIVED_EVENTS, 10000);
        if(ret < 0) {
            if(errno == EINTR){
                printf("epoll_wait was interrupted by signal\n");
                continue;
            }
            perror("epoll_wait error");
            finalizer(event->data.fd, epfd, event);
            return;
        }
        // timeout
        if (ret == 0) {
            printf("epoll_wait timeout\n");
            continue;
        }

        handle_events(epfd, recived_events, ret);
    }
}

void signal_handler(int signo){
    return;
}

void set_signal_handler(){
    struct sigaction act;
    sigemptyset(&act.sa_mask);
    act.sa_handler = signal_handler;
    act.sa_flags = SA_RESTART;

    int ret = sigaction(SIGUSR1, &act, NULL);
    if(ret != 0){
        perror("sigaction error");
        exit(1);
    }
}

int main() {
    set_signal_handler();
    boot_server();
    return 0;
}