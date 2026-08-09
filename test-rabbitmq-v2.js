#!/usr/bin/env node

const amqp = require('amqplib');
const crypto = require('crypto');

async function testRabbitMQ() {
  const rabbitmqURL = 'amqp://indieclaw:secretpass@localhost:5672/';
  let conn;

  try {
    console.log('🔗 Connecting to RabbitMQ...');
    conn = await amqp.connect(rabbitmqURL);
    const ch = await conn.createChannel();

    console.log('✅ Connected to RabbitMQ\n');

    // Verify queues exist
    console.log('📋 Verifying queues...');
    await ch.assertQueue('orchestrator.requests', { durable: true });
    await ch.assertQueue('orchestrator.responses', {
      durable: true,
      arguments: { 'x-expires': 3600000 }
    });
    console.log('   ✓ Queues ready\n');

    // Create unique test request
    const correlationId = `test_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`;
    const testRequest = {
      correlationId,
      phoneNumber: '919361315379',
      message: 'What is 2+2?',  // Simple question
      timestamp: Date.now(),
      idempotencyKey: `test_${Date.now()}`,
      images: []
    };

    // IMPORTANT: Start listening BEFORE sending message
    console.log('👂 Starting listener for responses...');
    let responseReceived = false;
    const responsePromise = new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Response timeout after 60 seconds'));
      }, 60000);

      ch.consume('orchestrator.responses', (msg) => {
        if (msg) {
          try {
            const response = JSON.parse(msg.content.toString());

            if (response.correlationId === correlationId) {
              clearTimeout(timeout);
              responseReceived = true;
              ch.ack(msg);
              console.log('   ✓ Listener received response for our correlationId\n');
              resolve(response);
            } else {
              // Not for us, requeue
              ch.nack(msg, false, true);
            }
          } catch (e) {
            ch.ack(msg);
          }
        }
      }, { noAck: false });
    });

    // Now send the request
    console.log('📤 Sending test request:');
    console.log(`   Correlation ID: ${correlationId}`);
    console.log(`   Phone: ${testRequest.phoneNumber}`);
    console.log(`   Message: ${testRequest.message}\n`);

    ch.sendToQueue(
      'orchestrator.requests',
      Buffer.from(JSON.stringify(testRequest)),
      {
        persistent: true,
        contentType: 'application/json',
        correlationId: correlationId
      }
    );

    console.log('✅ Request published to orchestrator.requests queue\n');
    console.log('⏳ Waiting for response (timeout: 60 seconds)...\n');

    try {
      const response = await responsePromise;

      console.log('📬 Response received!\n');
      console.log(`   Status: ${response.status || 'N/A'}`);
      console.log(`   Correlation ID: ${response.correlationId}`);

      if (response.result) {
        const resultPreview = typeof response.result === 'string'
          ? response.result.substring(0, 500)
          : JSON.stringify(response.result).substring(0, 500);
        console.log(`   Result: ${resultPreview}\n`);
      }

      console.log('🎉 SUCCESS: Adiyan system is working end-to-end!');
      console.log('\n✨ The orchestrator successfully:');
      console.log('   1. Received the message from RabbitMQ');
      console.log('   2. Processed it through the 7-stage pipeline');
      console.log('   3. Generated a response');
      console.log('   4. Published it back to orchestrator.responses');
      console.log('   5. We received it here');

    } catch (timeoutErr) {
      console.log(`❌ ${timeoutErr.message}`);
      console.log('\n⚠️  Troubleshooting:');
      console.log('   • Is orchestrator running? ./adiyan-logs.sh status');
      console.log('   • Check orchestrator logs: ./adiyan-logs.sh follow orchestrator');
      console.log('   • Check it\'s consuming requests: grep "Consuming" logs/orchestrator.log');
      process.exit(1);
    }

  } catch (error) {
    console.error('❌ Error:', error.message);
    if (error.message.includes('ECONNREFUSED')) {
      console.error('\n❌ RabbitMQ is not running on localhost:5672');
      console.error('   Start with: ./adiyan-start.sh');
    }
    process.exit(1);
  } finally {
    if (conn) {
      await conn.close();
    }
  }
}

testRabbitMQ();
