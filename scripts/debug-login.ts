import * as forge from 'node-forge';
import { createHttpClient } from '../src/client/http';

const USER_AGENT = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36';

async function main() {
  const client = createHttpClient();

  // 1. Init session
  console.log('1. Init session...');
  const sessionResp = await client.fetch('https://www.dhlottery.co.kr/login', {
    headers: { 'User-Agent': USER_AGENT },
  });
  console.log('Session status:', sessionResp.status);
  console.log('Cookies:', Object.keys(client.cookies));

  // 2. Get RSA key
  console.log('\n2. Get RSA key...');
  const rsaResp = await client.fetch('https://www.dhlottery.co.kr/login/selectRsaModulus.do', {
    headers: { 'User-Agent': USER_AGENT },
  });
  const rsaText = await rsaResp.text('utf-8');
  console.log('RSA response:', rsaText.substring(0, 500));
  const rsaData = JSON.parse(rsaText) as { publicKeyModulus: string; publicKeyExponent: string };
  console.log('RSA modulus length:', rsaData.publicKeyModulus?.length);

  if (!rsaData.publicKeyModulus) {
    console.log('No RSA key - server may be blocking');
    return;
  }

  // 3. Encrypt credentials
  const n = new forge.jsbn.BigInteger(rsaData.publicKeyModulus, 16);
  const e = new forge.jsbn.BigInteger(rsaData.publicKeyExponent, 16);
  const publicKey = forge.pki.setRsaPublicKey(n, e);

  const encrypt = (text: string) => {
    const encrypted = publicKey.encrypt(text, 'RSAES-PKCS1-V1_5');
    return forge.util.bytesToHex(encrypted);
  };

  const userId = process.env.USER_ID;
  const password = process.env.PASSWORD;
  if (!userId || !password) {
    throw new Error('Missing USER_ID or PASSWORD in environment');
  }

  const encryptedId = encrypt(userId);
  const encryptedPw = encrypt(password);

  // 4. Login
  console.log('\n3. Login...');
  const loginResp = await client.fetch('https://www.dhlottery.co.kr/login/securityLoginCheck.do', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'User-Agent': USER_AGENT,
    },
    body: new URLSearchParams({
      userId: encryptedId,
      password: encryptedPw,
      check: 'on',
      newsEventYn: '',
    }).toString(),
    redirect: 'manual',
  });

  console.log('Status:', loginResp.status);
  console.log('Location:', loginResp.headers.get('Location'));

  const text = await loginResp.text('utf-8');
  console.log('\nResponse body (first 1000 chars):');
  console.log(text.substring(0, 1000));
}

main().catch(console.error);
