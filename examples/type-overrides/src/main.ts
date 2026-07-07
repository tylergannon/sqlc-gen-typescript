import type { Sql } from "postgres";

import { createUser, getUser, updateUserBirthday } from "./db/query_sql";
import { emailAddress, strToDateTime, uuid } from "$lib/model/types";

export async function runOverrideExample(sql: Sql): Promise<void> {
  const userID = uuid("00000000-0000-4000-8000-000000000001");

  await createUser(sql, {
    id: userID,
    favoriteTeamId: null,
    email: emailAddress("ada@example.com"),
    birthday: strToDateTime("1815-12-10T00:00:00.000Z"),
    invitedAt: null,
    loginCount: 1,
    quotaBytes: null,
    metadata: { role: "admin", enabled: true },
  });

  const user = await getUser(sql, { id: userID });
  if (user !== null) {
    const nextBirthday = strToDateTime(user.birthday.toISOString());
    await updateUserBirthday(sql, {
      id: user.id,
      birthday: nextBirthday,
    });
  }
}
