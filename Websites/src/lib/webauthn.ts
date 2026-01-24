export interface WebAuthnCredential {
  id: string;
  rawId: string;
  type: string;
  response: Record<string, unknown>;
}

const base64urlToBuffer = (value: string): ArrayBuffer => {
  const padded = value.replace(/-/g, '+').replace(/_/g, '/');
  const decoded = atob(padded.padEnd(padded.length + (4 - (padded.length % 4)) % 4, '='));
  const buffer = new Uint8Array(decoded.length);
  for (let i = 0; i < decoded.length; i += 1) {
    buffer[i] = decoded.charCodeAt(i);
  }
  return buffer.buffer;
};

const bufferToBase64url = (buffer: ArrayBuffer): string => {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  bytes.forEach((b) => {
    binary += String.fromCharCode(b);
  });
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
};

export const isWebAuthnSupported = (): boolean =>
  typeof window !== 'undefined' && !!window.PublicKeyCredential;

export const prepareCreationOptions = (
  options: PublicKeyCredentialCreationOptions
): PublicKeyCredentialCreationOptions => ({
  ...options,
  challenge: base64urlToBuffer(options.challenge as unknown as string),
  user: {
    ...options.user,
    id: base64urlToBuffer(options.user.id as unknown as string),
  },
  excludeCredentials: options.excludeCredentials?.map((cred) => ({
    ...cred,
    id: base64urlToBuffer(cred.id as unknown as string),
  })),
});

export const prepareRequestOptions = (
  options: PublicKeyCredentialRequestOptions
): PublicKeyCredentialRequestOptions => ({
  ...options,
  challenge: base64urlToBuffer(options.challenge as unknown as string),
  allowCredentials: options.allowCredentials?.map((cred) => ({
    ...cred,
    id: base64urlToBuffer(cred.id as unknown as string),
  })),
});

export const serializeRegistration = (credential: Credential): WebAuthnCredential => {
  const publicKey = credential as PublicKeyCredential;
  const response = publicKey.response as AuthenticatorAttestationResponse;
  return {
    id: publicKey.id,
    rawId: bufferToBase64url(publicKey.rawId),
    type: publicKey.type,
    response: {
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      attestationObject: bufferToBase64url(response.attestationObject),
    },
  };
};

export const serializeAuthentication = (credential: Credential): WebAuthnCredential => {
  const publicKey = credential as PublicKeyCredential;
  const response = publicKey.response as AuthenticatorAssertionResponse;
  return {
    id: publicKey.id,
    rawId: bufferToBase64url(publicKey.rawId),
    type: publicKey.type,
    response: {
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      authenticatorData: bufferToBase64url(response.authenticatorData),
      signature: bufferToBase64url(response.signature),
      userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : null,
    },
  };
};
