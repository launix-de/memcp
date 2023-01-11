#include "HttpServer.hpp"
#include <iostream>
#include <string>
#include <cstdlib>

void printHelp()
{
    std::cout << "Usage: HttpServer -p [PORT] --secure -c [IP:PORT] --help" << std::endl;
    std::cout << "-p [PORT], --port [PORT]\tSpecifies the port to listen on" << std::endl;
    std::cout << "--secure\t\t\tEnables HTTPS (default is HTTP)" << std::endl;
    std::cout << "-c [IP:PORT], --connect [IP:PORT]\tSpecifies a connection to another node" << std::endl;
    std::cout << "-h, --help\t\t\tPrints this help message" << std::endl;
}
    
int main(int argc, char *argv[])
{   
    int port = 3877;
    std::string connectIP;
    int connectPort = 3877;
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
        else if (arg == "-c" || arg == "--connect")
        {
            std::string connectIP_and_Port = argv[i + 1];
            std::string connectIP = connectIP_and_Port.substr(0, connectIP_and_Port.find(":"));
            int connectPort = std::atoi(connectIP_and_Port.substr(connectIP_and_Port.find(":") + 1).c_str());
        }
        else if (arg == "-h" || arg == "--help")
        { 
            printHelp(); 
            return 0;
        }
    }   

    // connectIP, connectPort
    HttpServer server(port, isSecure);
    server.start();

    return 0;
}
