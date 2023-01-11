#include <string>
#include <iostream>
#include <uv.h>

class HttpServer {
 public:
  HttpServer(int port, bool isSecure)
    : port_(port),
      isSecure_(isSecure) {
  }

  void start() {
    uv_loop_t* loop = uv_default_loop();
    uv_tcp_t server;
    uv_tcp_init(loop, &server);
    uv_ip4_addr("0.0.0.0", port_, &addr_);
    uv_tcp_bind(&server, (const struct sockaddr*)&addr_, 0);

    uv_listen((uv_stream_t*) &server, 128, onConnection);

    uv_run(loop, UV_RUN_DEFAULT);
  }

 private:
  static void onConnection(uv_stream_t* server, int status) {
    if (status < 0) {
      std::cerr << "Error on connection: " << uv_strerror(status);
      return;
    }

    uv_tcp_t* client = (uv_tcp_t*)malloc(sizeof(uv_tcp_t));
    uv_tcp_init(uv_default_loop(), client);
    if (uv_accept(server, (uv_stream_t*)client) == 0) {
      uv_read_start((uv_stream_t*)client, onAlloc, onRead);
    } else {
      uv_close((uv_handle_t*)client, NULL);
    }
  }

  void onSecureConnection(uv_stream_t* server, int status) {
    // TODO: Implement secure connection
  }

  static void onRead(uv_stream_t* client, ssize_t nread, const uv_buf_t* buf) {
    if (nread < 0) {
      if (nread != UV_EOF)
        std::cerr << "Error on read: " << uv_strerror(nread);
      uv_close((uv_handle_t*)client, NULL);
    } else if (nread > 0) {
      std::string response = "Hello\n";
      uv_write_t* writeReq = (uv_write_t*)malloc(sizeof(uv_write_t));
      uv_buf_t resBuf = uv_buf_init(const_cast<char*>(response.data()), response.size());
      uv_write(writeReq, client, &resBuf, 1, NULL);
    }

    free(buf->base);
  }

  static void onAlloc(uv_handle_t* client, size_t suggestedSize, uv_buf_t* buf) {
    buf->base = (char*)malloc(suggestedSize);
    buf->len = suggestedSize;
  }

  int port_;
  bool isSecure_;
  struct sockaddr_in addr_;
};
