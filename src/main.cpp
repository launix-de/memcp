#include "HttpServer.hpp"
#include <iostream>
#include <string>
#include <cstdlib>

void printHelp()
{
    std::cout << "Usage: HttpServer -p [PORT] --secure --help" << std::endl;
    std::cout << "-p [PORT], --port [PORT]\tSpecifies the port to listen on" << std::endl;
    std::cout << "--secure\t\t\tEnables HTTPS (default is HTTP)" << std::endl;
    std::cout << "-h, --help\t\t\tPrints this help message" << std::endl;
}

int main(int argc, char *argv[])
{
    int port = 3877;
    bool isSecure = false;

    for (int i = 0; i < argc; i++)
    {
        std::string arg = argv[i];

        if (arg == "-p" || arg == "--port")
        {
            port = std::atoi(argv[i + 1]);
        }
        else if (arg == "--secure")
        {
            isSecure = true;
        }
        else if (arg == "-h" || arg == "--help")
        {
            printHelp();
            return 0;
        }
    }

    HttpServer server(port, isSecure);
    server.open();

    return 0;
}
