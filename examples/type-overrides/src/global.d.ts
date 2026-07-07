import type { DateTime as ImportedDateTime } from "$lib/model/types";

declare global {
  type DateTime = ImportedDateTime;
}

export {};
