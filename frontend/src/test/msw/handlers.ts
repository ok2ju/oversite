import { http, HttpResponse } from "msw"

export const handlers = [
  http.get("/api/v1/auth/me", () => {
    return HttpResponse.json({
      user_id: "test-user-id",
      faceit_id: "test-faceit-id",
      nickname: "TestPlayer",
    })
  }),
]
