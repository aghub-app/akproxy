import assert from "node:assert/strict";
import { test } from "node:test";
import type { Status } from "../lib/desktop.ts";
import { serviceBarNotice } from "./service-bar.ts";

function status(patch: Partial<Status>): Status {
  return {
    running: false,
    action: "start",
    address: "http://127.0.0.1:8317",
    allInterfaces: false,
    restartRequired: false,
    savedAddress: "http://127.0.0.1:9000",
    error: "",
    ...patch,
  };
}

test("a calm service has no second line", () => {
  assert.equal(serviceBarNotice(undefined), "");
  assert.equal(serviceBarNotice(status({ running: true, action: "stop" })), "");
});

test("restart names the address in use and the address after restart", () => {
  assert.equal(
    serviceBarNotice(status({ running: true, action: "restart", restartRequired: true })),
    "客户端仍使用 http://127.0.0.1:8317。重启后改为 http://127.0.0.1:9000。",
  );
});

test("a failure replaces the restart line", () => {
  assert.equal(
    serviceBarNotice(status({
      error: "端口被占用",
      restartRequired: true,
      running: true,
      action: "restart",
    })),
    "端口被占用",
  );
});
