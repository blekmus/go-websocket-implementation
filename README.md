# go-websocket-implementation

Decided to learn how websockets work by just implementing the RFC 6455 spec from scratch. I know there are libraries that do this already and to a much better degree too, but I learn better by doing so here we are.

### A little troubleshooting

Everything started smoothly at first, I built a TCP server using `net` to listen to requests and print out whatever request it got into the console.

I sent a few HTTP requests and saw all the headers and bodies printed out; as expected. However when I tested with a websocket type request in the API testing apps I used, everything fell apart. The server logs showed gibberish, and the client apps I used, from insomnia to postman all failed to connect.

Websockets are supposed to send a GET request to initiate the handshake before upgrading to the bi-directional channel so I was confused. Where is it? And why can't I decode whatever is being picked up by the server?

I debugged endlessly, certain it was a bug in my code. Frustrated, I reread the specification. That’s when it hit me. All, which was more than half a dozen clients, I used defaulted to `wss://`, the encrypted WebSocket protocol rather than `ws://`

My server wasn’t set up for TLS, so the *gibberish* was encrypted data. Once I switched the client to `ws://`, everything worked perfectly. Sometimes the issues you face are right in front of you but you don't know its wrong because you don't know what to look at. I'll forever be weary of extra s's at the end now.