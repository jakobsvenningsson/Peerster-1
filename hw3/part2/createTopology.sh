go build

./part2 -UIPort=12345 -gossipAddr=127.0.0.1:5001 -name=A -peers=127.0.0.1:5002, 127.0.0.1:5003 -rtimer=60 -GUIPort=8080 > file1 &
./part2 -UIPort=12346 -gossipAddr=127.0.0.1:5002 -name=B -peers=127.0.0.1:5001, 127.0.0.1:5003 -rtimer=60 -GUIPort=8081 > file2 &
./part2 -UIPort=12347 -gossipAddr=127.0.0.1:5003 -name=C -peers=127.0.0.1:5001, 127.0.0.1:5002,127.0.0.1:5004 -rtimer=60 -GUIPort=8082 > file3 &
./part2 -UIPort=12348 -gossipAddr=127.0.0.1:5004 -name=D -peers=127.0.0.1:5003, 127.0.0.1:5006  -rtimer=60 -GUIPort=8083 > file4 &
./part2 -UIPort=12349 -gossipAddr=127.0.0.1:5005 -name=E -peers=127.0.0.1:5003, 127.0.0.1:5006 -rtimer=60 -GUIPort=8084 > file5 &
./part2 -UIPort=12340 -gossipAddr=127.0.0.1:5006 -name=F -peers=127.0.0.1:5004, 127.0.0.1:5005 -rtimer=60 -GUIPort=8085 > file6 &
