# CMakeLists.txt

cmake_minimum_required (VERSION 3.0)
project (cpdb)

include_directories (${CMAKE_SOURCE_DIR}/src)
add_executable (cpdb)
target_link_libraries (cpdb -llmdb -luv)

# Add files from src
file (GLOB SOURCE_FILES "src/*.cpp" "src/*.hpp")
target_sources (cpdb PRIVATE ${SOURCE_FILES})
target_include_directories (cpdb PRIVATE ${CMAKE_SOURCE_DIR}/src)
