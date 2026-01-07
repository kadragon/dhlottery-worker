/**
 * Logger utility
 */

import { DEBUG } from '../constants';

type LogDetails = Record<string, unknown>;
type LogLevel = 'debug' | 'info' | 'error';

function formatEntry(level: LogLevel, message: string, details?: LogDetails): string {
  return JSON.stringify({
    level,
    message,
    ...details,
    timestamp: new Date().toISOString(),
  });
}

export const logger = {
  debug(message: string, details?: LogDetails): void {
    if (!DEBUG) return;
    const output = formatEntry('debug', message, details);
    console.log(output);
  },
  info(message: string, details?: LogDetails): void {
    const output = formatEntry('info', message, details);
    console.log(output);
  },
  error(message: string, details?: LogDetails): void {
    const output = formatEntry('error', message, details);
    console.error(output);
  },
};
