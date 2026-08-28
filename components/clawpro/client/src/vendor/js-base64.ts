function ensureBinary(input: string) {
  try {
    return atob(input);
  } catch {
    return "";
  }
}

function encodeUtf8(input: string) {
  return btoa(unescape(encodeURIComponent(input)));
}

function decodeUtf8(input: string) {
  const binary = ensureBinary(input);
  return decodeURIComponent(escape(binary));
}

export const Base64 = {
  encode(input: string) {
    return encodeUtf8(input);
  },
  decode(input: string) {
    return decodeUtf8(input);
  },
};

export default Base64;
