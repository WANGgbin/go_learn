#include<cstdio>
#include<sys/socket.h>

using namespace std;


int main() {
    int sock_fd = socket(AF_INET, SOCK_STREAM, 0);
    if(sock_fd < 0) {
        perror("create client socket error");
        return 0;
    }

    int conn_ret = connect();
    return 0;
}