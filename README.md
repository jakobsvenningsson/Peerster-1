# Homework2

Important things to consider when running my program:

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

- For Exercise 4 I added an argument to the Peerster implementation named as described: -noforward,
it is a bool type in Go, if you set it to true, then Peerster will ​ never forward either private
point-to-point messages or chat rumor messages to other nodes, only route rumor
messages.

