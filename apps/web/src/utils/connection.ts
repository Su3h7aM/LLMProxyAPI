import { MANAGEMENT_API_PREFIX } from './constants';

/**
 * Same-origin management base: the panel ships inside the proxy binary,
 * so the API is always the page origin plus the v8 prefix. There is no
 * custom or remote base, no detection, and no normalization.
 */
export const getSameOriginApiBase = (): string => {
  if (typeof window === 'undefined' || !window.location?.origin) {
    return '';
  }
  return window.location.origin.replace(/\/+$/, '');
};

export const getManagementApiUrl = (): string => {
  const base = getSameOriginApiBase();
  return base ? `${base}${MANAGEMENT_API_PREFIX}` : MANAGEMENT_API_PREFIX;
};
