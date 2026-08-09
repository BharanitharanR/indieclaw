#!/usr/bin/env node

const amqp = require('amqplib');
const crypto = require('crypto');

async function testRabbitMQ() {
  const rabbitmqURL = 'amqp://indieclaw:secretpass@localhost:5672/';

  try {
    console.log('🔗 Connecting to RabbitMQ...');
    const conn = await amqp.connect(rabbitmqURL);
    const ch = await conn.createChannel();

    console.log('✅ Connected to RabbitMQ\n');

    // Declare queues with TTL matching orchestrator configuration
    console.log('📋 Verifying queues...');
    await ch.assertQueue('orchestrator.requests', { durable: true });
    await ch.assertQueue('orchestrator.responses', {
      durable: true,
      arguments: { 'x-expires': 3600000 }  // 1 hour TTL matching orchestrator
    });

    console.log('   ✓ Queues ready\n');

    // Create test request
    const correlationId = `test_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`;
    const testRequest = {
      correlationId,
      phoneNumber: '919361315379',
      message: 'How can I improve my communication skills?',
      timestamp: Date.now(),
      idempotencyKey: `test_${Date.now()}`,
      images: []
    };

    console.log('📤 Sending test request:');
    console.log(`   Correlation ID: ${correlationId}`);
    console.log(`   Phone: ${testRequest.phoneNumber}`);
    console.log(`   Message: ${testRequest.message}\n`);

    // Send request to orchestrator
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

    // Wait for response with timeout
    console.log('⏳ Waiting for response (timeout: 35 seconds)...\n');

    const responsePromise = new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Response timeout after 35 seconds'));
      }, 35000);

      let consumerTag;

      ch.consume('orchestrator.responses', (msg) => {
        if (msg) {
          try {
            const response = JSON.parse(msg.content.toString());

            if (response.correlationId === correlationId) {
              clearTimeout(timeout);
              ch.ack(msg);
              if (consumerTag) ch.cancel(consumerTag).catch(() => {});
              resolve(response);
            } else {
              // Not our message, requeue it
              ch.nack(msg, false, true);
            }
          } catch (e) {
            ch.ack(msg);
          }
        }
      }, { noAck: false }).then(tag => {
        consumerTag = tag.consumerTag;
      });
    });

    try {
      const response = await responsePromise;

      console.log('📬 Response received:\n');
      console.log(`   Status: ${response.status}`);
      console.log(`   Correlation ID: ${response.correlationId}`);
      console.log(`   Processing Time: ${response.processingTimeMs || 'N/A'}ms\n`);

      if (response.status === 'success') {
        console.log('✅ Response Result:');
        const resultPreview = typeof response.result === 'string'
          ? response.result.substring(0, 300)
          : JSON.stringify(response.result).substring(0, 300);
        console.log(`   ${resultPreview}...\n`);
        console.log('🎉 SUCCESS: Adiyan system is working correctly!');
      } else {
        console.log('❌ Error:');
        console.log(`   ${response.error || 'Unknown error'}\n`);
        console.log('⚠️  ISSUE: Orchestrator returned error');
      }

    } catch (timeoutErr) {
      console.log(`❌ ${timeoutErr.message}`);
      console.log('⚠️  No response received - orchestrator may not be processing requests');
      console.log('\nTroubleshooting:');
      console.log('1. Check if orchestrator is running: ./adiyan-logs.sh status');
      console.log('2. View orchestrator logs: ./adiyan-logs.sh follow orchestrator');
      console.log('3. Verify RabbitMQ is running: lsof -i :5672');
    }

    await conn.close();

  } catch (error) {
    console.error('❌ Connection Error:', error.message);
    if (error.message.includes('ECONNREFUSED')) {
      console.error('\nRabbitMQ is not running on localhost:5672');
      console.error('Start it with: ./adiyan-start.sh');
    } else if (error.message.includes('transient_nonexcl_queues')) {
      console.error('\nRabbitMQ configuration issue: transient queues are disabled');
      console.error('This may require RabbitMQ configuration changes');
    }
    process.exit(1);
  }
}

testRabbitMQ();
