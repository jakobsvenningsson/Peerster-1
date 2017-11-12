# Homework2 README

Important things to consider when running my Peerster:

**To run the CLI(client):**
-If we want to send a Rumor Message to the node that listens to the port UIPort:

./client/client -UIPort=12346 -Dest="" -msg="testMessage"

-If we want to send a Private Message to the node that listens to the port UIPort with destination specified in the argument Dest:

./client/client -UIPort=12346 -Dest="A" -msg="testMessage"


**to run the Peerster:**
(assuming the user is in the directory /part1)

go build
- ./part1 -UIPort=12345 -gossipAddr=127.0.0.1:5001 -name=A -peers=127.0.0.1:5002 -rtimer=60 -GUIPort=0

notes: if we want to run the GUI we must give a proper value to the argument GUIPort(e.g. 8080,8081)
if we want to give more than one peers as arguments we should separate them with '_',
(e.g. -peers=127.0.0.1:5002_127.0.0.1:5003)


###Implementation Details:


**Corrections on the Gossip Protocol**

I distinguish the Status Packets that come from AntiEntropy and the others that come as a response from a RumorMessage
using the channel **msgChnACK** and the map **waitForACK**

### Part1

In the routing.go I implement the functionality of the routing protocol.
In the routingtable.go  I implement the routing table according to the specs.

For the rtimer, I think we use to synchronize the route rumor messaging,
and I would use it in routing.go in waitForRumorMessage.
I have it commented since the tests you provided to us sometimes failed, because of the time of sleep command.
e.g. when I changed the argument of the sleep in the test scripts from 5 to 7, they succeeded.
So I the the code for the rtimer commented and since the router does not wait any time between route rumors, tests succeed.

Regarding the GUI, it works well according to specs.
The only thing I did not fix is that I print in the chat box the messages as the vector clock returns them,
not according to any sorting mechanism. Of course, in the backend everything works well regarding the sequencing of messages,
based on the msgID from exercise1.

### Part2

Here I did not manage to run on the VM.
The reason is that, when I tried to "go get" the gorilla mux and the protobuf from the github,
runtime errors appeared and as a result when I tried to run the tests, these errors happened again.
It's Sunday evening and I do not want to spam with emails, I will ask a TA until the grading day.
So I tested my implementation locally with the same arguments you provide in the test scripts.

- For Exercise 4:
 I added an argument to the Peerster implementation named as described: -noforward,
it is a bool type in Go, if you set it to true, then Peerster will ​ never forward either private
point-to-point messages or chat rumor messages to other nodes, only route rumor
messages.

To test this locally I created 3 nodes:
(assuming the user is in the directory /part2)

./part2 -UIPort=12345 -gossipAddr=127.0.0.1:5001 -name=A -peers=127.0.0.1:5002 -GUIPort=8080

./part2 -UIPort=12346 -gossipAddr=127.0.0.1:5002 -name=B -peers=127.0.0.1:5003_127.0.0.1:5001 -noforward

./part2 -UIPort=12347 -gossipAddr=127.0.0.1:5003 -name=C -peers=127.0.0.1:5002 -GUIPort=8081

In this case node B acts as NodePub and when I try to send private messages and Rumor chat messages from A or C they reach B(NodePub), but B does not forward them.
In the GUI in the node Identifiers list of node A there is not only the origin of B, but also of C,
which is right. But in the list of known peers node A does not have the IP of C and vice versa.

-For Exercise 5:

In this case with the same commands as above I added some code according to the specs and A can send directly messages to C. The basic difference lies in line 255 of gossiper2.go

-For Exercise 6:









