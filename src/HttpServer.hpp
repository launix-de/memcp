#ifndef HTTPSERVER_HPP
#define HTTPSERVER_HPP

#include <string>
#include <iostream>
#include <boost/asio.hpp>

using namespace boost::asio;

class HttpServer
{
public:
    HttpServer(int port, bool isSecure);

    void open();

private:
    int m_port;
    bool m_isSecure;
};

#endif
