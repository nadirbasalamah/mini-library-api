// Paste into the login request's Post-response (Tests) script in Postman.
// The script checks JWT claims but does not verify the JWT signature.

let body;
try {
  body = pm.response.json();
} catch (error) {
  body = undefined;
}

pm.test("Login returns HTTP 200", function () {
  pm.response.to.have.status(200);
});

pm.test("Login response matches the expected contract", function () {
  pm.expect(body, "response body should be valid JSON").to.be.an("object");
  pm.expect(body).to.have.keys("status", "message", "data");
  pm.expect(body.status).to.eql("success");
  pm.expect(body.message).to.eql("login success");
  pm.expect(body.data).to.be.a("string").and.not.empty;
});

function decodeBase64Url(value) {
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, "=");
  return atob(padded);
}

let header;
let claims;
let tokenParts;
if (body && typeof body.data === "string") {
  tokenParts = body.data.split(".");
  if (tokenParts.length === 3) {
    try {
      header = JSON.parse(decodeBase64Url(tokenParts[0]));
      claims = JSON.parse(decodeBase64Url(tokenParts[1]));
    } catch (error) {
      header = undefined;
      claims = undefined;
    }
  }
}

pm.test("Login data is a three-part JWT using HS256", function () {
  pm.expect(tokenParts, "JWT should contain three segments").to.have.length(3);
  pm.expect(header, "JWT header should be valid JSON").to.be.an("object");
  pm.expect(header.alg).to.eql("HS256");
  pm.expect(tokenParts[2], "JWT signature segment should not be empty").to.not.be.empty;
});

pm.test("JWT contains the expected user claims", function () {
  pm.expect(claims, "JWT payload should be valid JSON").to.be.an("object");
  pm.expect(claims.id).to.be.a("number").and.to.be.greaterThan(0);
  pm.expect(claims.role).to.eql("user");
  pm.expect(claims.exp).to.be.a("number").and.to.be.greaterThan(Math.floor(Date.now() / 1000));

  const expectedUserId = pm.environment.get("expectedUserId");
  if (expectedUserId) {
    pm.expect(claims.id).to.eql(Number(expectedUserId));
  }
});

if (body && typeof body.data === "string" && body.data.length > 0) {
  pm.collectionVariables.set("authToken", body.data);
}
