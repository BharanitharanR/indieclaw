#!/usr/bin/env node

const amqp = require('amqplib');
const crypto = require('crypto');

async function captureResponse() {
  const rabbitmqURL = 'amqp://indieclaw:secretpass@localhost:5672/';
  let conn;

  try {
    console.log('🔗 Connecting to RabbitMQ...');
    conn = await amqp.connect(rabbitmqURL);
    const ch = await conn.createChannel();

    console.log('✅ Connected\n');

    // Verify queues
    await ch.assertQueue('orchestrator.requests', { durable: true });
    await ch.assertQueue('orchestrator.responses', {
      durable: true,
      arguments: { 'x-expires': 3600000 }
    });

    // Create unique test request
    const correlationId = `capture_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`;
    const testRequest = {
      correlationId,
      phoneNumber: '919361315379',
      message: 'What are the key principles of effective leadership communication?',
      timestamp: Date.now(),
      idempotencyKey: `test_${Date.now()}`,
      images: []
    };

    console.log('📤 Sending request:');
    console.log(`   Correlation ID: ${correlationId}`);
    console.log(`   Message: "${testRequest.message}"\n`);

    // Start listening BEFORE sending
    let gotResponse = false;
    const responsePromise = new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Response timeout'));
      }, 300000);  // 5 minute timeout for thorough processing

      ch.consume('orchestrator.responses', (msg) => {
        if (msg) {
          try {
            const response = JSON.parse(msg.content.toString());
            if (response.correlationId === correlationId) {
              clearTimeout(timeout);
              gotResponse = true;
              ch.ack(msg);
              resolve(response);
            } else {
              ch.nack(msg, false, true);
            }
          } catch (e) {
            ch.ack(msg);
          }
        }
      }, { noAck: false });
    });

    // Send request
    ch.sendToQueue(
      'orchestrator.requests',
      Buffer.from(JSON.stringify(testRequest)),
      {
        persistent: true,
        contentType: 'application/json',
        correlationId: correlationId
      }
    );

    console.log('✅ Request published\n');
    console.log('⏳ Waiting for response (up to 5 minutes)...\n');

    try {
      const response = await responsePromise;

      console.log('═'.repeat(80));
      console.log('📬 RESPONSE RECEIVED');
      console.log('═'.repeat(80));
      console.log(`\n📋 Metadata:`);
      console.log(`   Correlation ID: ${response.correlationId}`);
      console.log(`   Status: ${response.status || 'N/A'}`);
      console.log(`   Length: ${response.result?.length || 0} characters\n`);

      console.log('📄 Response Text:');
      console.log('─'.repeat(80));
      if (response.result) {
        console.log(response.result);
      } else if (response.error) {
        console.log(`ERROR: ${response.error}`);
      } else {
        console.log(JSON.stringify(response, null, 2));
      }
      console.log('─'.repeat(80));

      console.log('\n✨ Response successfully captured!');

    } catch (err) {
      console.log(`❌ ${err.message}`);
      console.log('\nTroubleshooting:');
      console.log('  1. Check orchestrator: ./adiyan-logs.sh status');
      console.log('  2. View logs: ./adiyan-logs.sh follow orchestrator');
      process.exit(1);
    }

  } catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
  } finally {
    if (conn) {
      await conn.close();
    }
  }
}

captureResponse();
