/**
 * Real Purchase Test Script
 * WARNING: This will actually purchase lottery tickets and charge real money (5,000 KRW)
 */

import { config } from 'dotenv';
import { runWorkflow } from './src/index';
import type { WorkerEnv } from './src/index';

// Load environment variables from .dev.vars
config({ path: '.dev.vars' });

const env: WorkerEnv = {
  USER_ID: process.env.USER_ID || '',
  PASSWORD: process.env.PASSWORD || '',
  TELEGRAM_BOT_TOKEN: process.env.TELEGRAM_BOT_TOKEN || '',
  TELEGRAM_CHAT_ID: process.env.TELEGRAM_CHAT_ID || '',
};

async function main() {
  console.log('🚀 Starting real lottery purchase test...');
  console.log('⚠️  WARNING: This will charge 5,000 KRW to your account');
  console.log('');
  console.log('Environment check:');
  console.log(`- USER_ID: ${env.USER_ID ? '✓ Set' : '✗ Missing'}`);
  console.log(`- PASSWORD: ${env.PASSWORD ? '✓ Set' : '✗ Missing'}`);
  console.log(`- TELEGRAM_BOT_TOKEN: ${env.TELEGRAM_BOT_TOKEN ? '✓ Set' : '✗ Missing'}`);
  console.log(`- TELEGRAM_CHAT_ID: ${env.TELEGRAM_CHAT_ID ? '✓ Set' : '✗ Missing'}`);
  console.log('');

  if (!env.USER_ID || !env.PASSWORD || !env.TELEGRAM_BOT_TOKEN || !env.TELEGRAM_CHAT_ID) {
    console.error('❌ Missing required environment variables');
    process.exit(1);
  }

  try {
    console.log('🎯 Executing workflow...');
    await runWorkflow(env, new Date());
    console.log('');
    console.log('✅ Workflow completed successfully!');
    console.log('📱 Check your Telegram for notifications');
  } catch (error) {
    console.error('');
    console.error('❌ Workflow failed:');
    console.error(error);
    process.exit(1);
  }
}

main();
