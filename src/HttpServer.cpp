#include <string>
#include <iostream>
#include <boost/asio.hpp>
using namespace boost::asio;

#include "HttpServer.hpp"

HttpServer::HttpServer(int port, bool isSecure)
{
    m_port = port;
    m_isSecure = isSecure;
}

void HttpServer::open()
{
    io_service ios;
    ip::tcp::endpoint endpoint(ip::tcp::v4(), m_port);
    ip::tcp::acceptor acceptor(ios, endpoint);

    std::cout << "Listening for connections on port " << m_port << "..." << std::endl;

    while (true)
    {
        ip::tcp::socket socket(ios);
        acceptor.accept(socket);

        std::string message = "";
        if (m_isSecure)
        {
            message = "Securely accepted connection on port " + std::to_string(m_port);
        }
        else
        {
            message = "Accepted connection on port " + std::to_string(m_port);
        }

        std::cout << message << std::endl;
        write(socket, buffer(message));
    }
}
