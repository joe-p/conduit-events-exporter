import WebSocket from 'websocket';
// eslint-disable-next-line new-cap
const client = new WebSocket.client();


client.on('connectFailed', (error) => {
  console.error(`Connect Error: ${error.toString()}`);
});

client.on('connect', (connection) => {
  console.log('WebSocket Client Connected');

  connection.on('error', (error) => {
    console.error(`Connection Error: ${error.toString()}`);
  });

  connection.on('close', () => {
    console.log('Connection Closed');
  });

  connection.on('message', (message) => {
    if (message.type === 'utf8') {
      console.log(message.utf8Data)
    }
  });
});

client.connect('ws://127.0.0.1:3333/2');