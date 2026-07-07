export type UUID = string & { readonly __brand: "UUID" };

export type EmailAddress = string & { readonly __brand: "EmailAddress" };

export type DateTime = Date & { readonly __brand: "DateTime" };

export function strToDateTime(value: string): DateTime {
  return new Date(value) as DateTime;
}

export function uuid(value: string): UUID {
  return value as UUID;
}

export function emailAddress(value: string): EmailAddress {
  return value as EmailAddress;
}
