# API Test Cases

This catalog covers positive and negative cases for all 21 routes registered in `api/api.go`. Expected outcomes reflect the current handlers and middleware; where behavior is surprising or incomplete, it is called out under **Contract notes** rather than silently defining new behavior.

## Test conventions and fixtures

- Send requests to the exact method and path shown below. JSON requests use `Content-Type: application/json`.
- Use an in-process Fiber app, deterministic service stubs, and generated test JWTs. Do not call live PostgreSQL, Cloudinary, or Rajaongkir services.
- `member` means a valid JWT with a non-admin role; `admin` means a valid JWT with the admin role.
- For routes protected by JWT, also run the shared auth negatives:
  - **A1 — no token:** omit `Authorization`; current Fiber/JWT middleware returns `401` with `status: "error"` and `message: "Invalid or expired JWT"`.
  - **A2 — invalid/expired token:** send an invalid or expired bearer token; current JWT middleware returns `401` with `status: "error"` and `message: "Invalid or expired JWT"`.
  - **A3 — member on admin route:** use a valid member token; `VerifyAdmin` returns `403` with `message: "access denied"`.
- For service failures, configure the stub to return an error. Verify the endpoint-specific status and response below; no external service should be contacted.
- Unless stated otherwise, assert the response JSON `status`, `message`, and relevant `data` fields, not just the HTTP status.

## Authentication

### POST `/api/v1/auth/register`

| ID | Case | Request | Expected |
|---|---|---|---|
| AUTH-REG-P1 | Register valid user | `{"username":"reader1","email":"reader1@example.com","password":"StrongPass1!","address":"1 Main St","province_id":1,"city_id":2,"district_id":3}` | `201`; `status:"success"`, `message:"user registered"`, user data returned without password |
| AUTH-REG-N1 | Invalid JSON | Malformed JSON body | `400`; `status:"failed"`, `message:"invalid request"` |
| AUTH-REG-N2 | Required field missing or invalid email/password | Omit `username`, or use invalid email/password (less than 8 chars or missing required character classes) | `422`; `status:"failed"`, `message:"validation failed"` |
| AUTH-REG-N3 | Registration service failure | Valid request; stub returns error | `500`; `status:"failed"`, error message from `utils.GetErrorMessage` |

### POST `/api/v1/auth/login`

| ID | Case | Request | Expected |
|---|---|---|---|
| AUTH-LOGIN-P1 | Login successfully | `{"email":"reader1@example.com","password":"StrongPass1!"}`; auth stub returns user | `200`; `status:"success"`, `message:"login success"`, non-empty token in `data` |
| AUTH-LOGIN-N1 | Invalid JSON | Malformed JSON body | `400`; `status:"failed"`, `message:"invalid request"` |
| AUTH-LOGIN-N2 | Invalid/missing credentials shape | Missing email/password or invalid email | `422`; `status:"failed"`, `message:"validation failed"` |
| AUTH-LOGIN-N3 | Login service rejects credentials | Valid-shaped request; auth service returns error | `500`; `status:"failed"` and mapped error message |

## Profile

All profile routes require a valid JWT; run A1 and A2 for each.

### GET `/api/v1/profile`

| ID | Case | Request | Expected |
|---|---|---|---|
| PROFILE-GET-P1 | Read own profile | Member token; profile stub returns that user's profile | `200`; `status:"success"`, `message:"user data"`, profile in `data` |
| PROFILE-GET-N1 | Profile does not exist/service lookup fails | Member token; profile service returns error | `404`; `status:"failed"` and mapped error message |

### PATCH `/api/v1/profile/edit`

| ID | Case | Request | Expected |
|---|---|---|---|
| PROFILE-EDIT-P1 | Update profile without changing picture | Member token; valid form/JSON fields `username`, `email`, `address`; no file | `200`; `status:"success"`, `message:"profile updated"`; existing picture is preserved |
| PROFILE-EDIT-P2 | Replace profile picture | Member token; same valid fields and supported image file | `200`; updated picture data returned |
| PROFILE-EDIT-N1 | Invalid profile fields | Member token; omit required field or provide invalid email | `422`; validation failure response |
| PROFILE-EDIT-N2 | Profile lookup fails | Member token; user lookup returns error | `404`; `status:"failed"`, `message:"user not found"` |
| PROFILE-EDIT-N3 | Unsupported picture extension | Member token; upload a file with unsupported extension | `422`; `status:"failed"`, `message:"invalid file format"` |
| PROFILE-EDIT-N4 | Upload or old-picture deletion fails | Member token; supported file; uploader returns an error | `500`; upload failure or `message:"old file failed to delete"` |
| PROFILE-EDIT-N5 | Update service fails | Member token; valid fields; update stub returns error | `500`; `status:"failed"` and mapped error message |

## Categories

All category routes require a valid JWT; run A1 and A2 for each. Category create/update/delete also run A3.

### GET `/api/v1/categories`

| ID | Case | Request | Expected |
|---|---|---|---|
| CAT-LIST-P1 | List categories | Member token; optionally `?page=1&limit=10&sort=name%20asc&search=fiction` | `200`; `status:"success"`, `message:"all categories"`, paginated result in `data` |
| CAT-LIST-N1 | Category service fails | Member token; list stub returns error | `500`; `status:"failed"`, `message:"fetch category failed"` |

### GET `/api/v1/categories/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| CAT-GET-P1 | Read existing category | Member token; `GET /api/v1/categories/7` | `200`; `status:"success"`, `message:"category found"`, category in `data` |
| CAT-GET-N1 | Malformed ID | Member token; `GET /api/v1/categories/not-a-number` | `400`; `status:"failed"`, `message:"invalid id"` |
| CAT-GET-N2 | Category service fails/not found | Member token; lookup stub returns error | `500`; `status:"failed"` and mapped error message |

### POST `/api/v1/categories`

| ID | Case | Request | Expected |
|---|---|---|---|
| CAT-CREATE-P1 | Create category | Admin token; `{"name":"Fiction","description":"Fiction books"}` | `201`; `status:"success"`, `message:"category created"`, category in `data` |
| CAT-CREATE-N1 | Missing required field | Admin token; omit `name` or `description` | `422`; `status:"failed"`, `message:"validation failed"` |
| CAT-CREATE-N2 | Create service fails | Admin token; valid body; service returns error | `500`; `status:"failed"` and mapped error message |

### PATCH `/api/v1/categories/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| CAT-UPDATE-P1 | Update category | Admin token; `PATCH /api/v1/categories/7`, `{"name":"New name","description":"Updated description"}` | `200`; `status:"success"`, `message:"category updated"`, updated category in `data` |
| CAT-UPDATE-N1 | Malformed ID | Admin token; `PATCH /api/v1/categories/nope` with valid body | `400`; `status:"failed"`, `message:"invalid id"` |
| CAT-UPDATE-N2 | Invalid body | Admin token; omit required field | `422`; validation failure response |
| CAT-UPDATE-N3 | Update service fails | Admin token; valid ID/body; service returns error | `500`; `status:"failed"` and mapped error message |

### DELETE `/api/v1/categories/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| CAT-DELETE-P1 | Delete category | Admin token; `DELETE /api/v1/categories/7` | `200`; `status:"success"`, `message:"category deleted"` |
| CAT-DELETE-N1 | Malformed ID | Admin token; `DELETE /api/v1/categories/nope` | `400`; `status:"failed"`, `message:"invalid id"` |
| CAT-DELETE-N2 | Delete service fails | Admin token; valid ID; service returns error | `500`; `status:"failed"` and mapped error message |

## Books

All book routes require a valid JWT; run A1 and A2 for each. Book create/update/delete also run A3.

### GET `/api/v1/books`

| ID | Case | Request | Expected |
|---|---|---|---|
| BOOK-LIST-P1 | List books | Member token; optionally `?page=1&limit=10&sort=title%20asc&search=dune` | `200`; `status:"success"`, `message:"all books"`, paginated result in `data` |
| BOOK-LIST-N1 | Book service fails | Member token; list stub returns error | `500`; `status:"failed"`, `message:"fetch books failed"` |

### GET `/api/v1/books/category/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| BOOK-CAT-P1 | List books in category | Member token; `GET /api/v1/books/category/7?page=1&limit=5` | `200`; `status:"success"`, `message:"books by category"`, paginated result in `data` |
| BOOK-CAT-N1 | Malformed category ID | Member token; `GET /api/v1/books/category/nope` | `400`; `status:"failed"`, `message:"invalid id"` |
| BOOK-CAT-N2 | Service fails | Member token; category list stub returns error | `500`; `status:"failed"` and mapped error message |

### GET `/api/v1/books/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| BOOK-GET-P1 | Read existing book | Member token; `GET /api/v1/books/12` | `200`; `status:"success"`, `message:"book found"`, book in `data` |
| BOOK-GET-N1 | Malformed ID | Member token; `GET /api/v1/books/nope` | `400`; `status:"failed"`, `message:"invalid id"` |
| BOOK-GET-N2 | Lookup fails/not found | Member token; lookup stub returns error | `500`; `status:"failed"` and mapped error message |

### POST `/api/v1/books`

Use multipart form fields `title`, `description`, `publisher`, `year`, `stock`, `category_id`, and `file`.

| ID | Case | Request | Expected |
|---|---|---|---|
| BOOK-CREATE-P1 | Create book with image | Admin token; valid fields and supported image file | `201`; `status:"success"`, `message:"book created"`, book in `data` |
| BOOK-CREATE-N1 | Invalid/missing required fields | Admin token; omit a required field | `422`; validation failure response |
| BOOK-CREATE-N2 | Missing file | Admin token; valid fields but no `file` | `422`; `status:"failed"`, `message:"file not found"` |
| BOOK-CREATE-N3 | Unsupported file extension | Admin token; valid fields and unsupported file type | `422`; `status:"failed"`, `message:"invalid file format"` |
| BOOK-CREATE-N4 | Upload or create service fails | Admin token; supported file; configure uploader/service failure | `500`; `message:"upload failed"` or mapped service error |

### PATCH `/api/v1/books/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| BOOK-UPDATE-P1 | Update book without replacing image | Admin token; valid ID and required form fields; omit file | `200`; `status:"success"`, `message:"book updated"`; current image is retained |
| BOOK-UPDATE-P2 | Replace book image | Admin token; valid ID/fields and supported image file | `200`; updated book returned; replacement image recorded |
| BOOK-UPDATE-N1 | Malformed ID | Admin token; `PATCH /api/v1/books/nope` | `400`; `status:"failed"`, `message:"invalid id"` |
| BOOK-UPDATE-N2 | Invalid fields or file type | Admin token; omit required field or upload unsupported extension | `422`; validation failure or `message:"invalid file format"` |
| BOOK-UPDATE-N3 | Lookup, upload, old-image deletion, or update fails | Admin token; configure the corresponding dependency failure | `500`; endpoint-specific failure message (`upload failed`, `old image failed to delete`, or mapped service error) |

### DELETE `/api/v1/books/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| BOOK-DELETE-P1 | Delete book without image | Admin token; book exists and has no image | `200`; `status:"success"`, `message:"book deleted"` |
| BOOK-DELETE-P2 | Delete book and image | Admin token; book has image ID; uploader and service succeed | `200`; `status:"success"`, `message:"book deleted"` |
| BOOK-DELETE-N1 | Malformed ID | Admin token; `DELETE /api/v1/books/nope` | `400`; `status:"failed"`, `message:"invalid id"` |
| BOOK-DELETE-N2 | Lookup, image deletion, or book deletion fails | Admin token; configure one dependency to fail | `500`; endpoint-specific error (`image failed to delete` or mapped service error) |

## Rents

All rent routes require a valid JWT; run A1 and A2 for each. Rent list/update/delete also run A3.

### GET `/api/v1/rents`

| ID | Case | Request | Expected |
|---|---|---|---|
| RENT-LIST-P1 | Admin lists rents | Admin token; optionally `?page=1&limit=10&sort=id%20desc&search=jne` | `200`; `status:"success"`, `message:"all rents"`, rent list in `data` |
| RENT-LIST-N1 | Member cannot list all rents | Member token | `403`; `message:"access denied"` |
| RENT-LIST-N2 | Rent service fails | Admin token; list stub returns error | `500`; `status:"failed"`, `message:"fetch rents failed"` |

### GET `/api/v1/rents/me`

| ID | Case | Request | Expected |
|---|---|---|---|
| RENT-ME-P1 | Read own rents | Member token; rent stub returns rows for token user | `200`; `status:"success"`, `message:"user rents"`; only the authenticated user's ID is used |
| RENT-ME-N1 | Rent service fails | Member token; service returns error | `500`; `status:"failed"` and mapped error message |

### GET `/api/v1/rents/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| RENT-GET-P1 | Owner reads own rent | Member token whose user ID matches rent owner | `200`; `status:"success"`, `message:"rent found"` |
| RENT-GET-P2 | Admin reads another user's rent | Admin token | `200`; `status:"success"`, `message:"rent found"` |
| RENT-GET-N1 | Malformed ID | Valid member token; `GET /api/v1/rents/nope` | `400`; `status:"failed"`, `message:"invalid id"` |
| RENT-GET-N2 | Another member reads rent | Member token whose ID differs from rent owner | `403`; `status:"failed"`, `message:"access denied"` |
| RENT-GET-N3 | Rent lookup fails/not found | Valid token; rent lookup stub returns error | `500`; `status:"failed"` and mapped error message |

### POST `/api/v1/rents`

Valid JSON example: `{"books":[12],"duration":7,"courier":"jne"}`. Use a configured local user stub and Rajaongkir fake.

| ID | Case | Request | Expected |
|---|---|---|---|
| RENT-CREATE-P1 | Create rent | Member token; valid book IDs, `duration >= 1`, supported courier; user, fee, and create stubs succeed | `201`; `status:"success"`, `message:"rent created"` |
| RENT-CREATE-N1 | Invalid request | Member token; omit books/courier/duration, use duration `0`, or unsupported courier | `422`; `status:"failed"`, `message:"validation failed"` |
| RENT-CREATE-N2 | User lookup fails | Member token; user service returns error | `500`; `status:"failed"` and mapped error message |
| RENT-CREATE-N3 | Delivery fee lookup fails | Member token; Rajaongkir fake returns error | `500`; `status:"failed"` and fee error message |
| RENT-CREATE-N4 | Rent creation fails | Member token; validation, user, and fee steps succeed; rent service returns error | `500`; `status:"failed"` and mapped error message |

### PATCH `/api/v1/rents/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| RENT-UPDATE-P1 | Admin updates rent status | Admin token; valid ID; `{"status":"returned"}` | `200`; `status:"success"`, `message:"rent updated"`, updated rent in `data` |
| RENT-UPDATE-N1 | Member is forbidden | Member token; valid ID/body | `403`; `message:"access denied"` |
| RENT-UPDATE-N2 | Malformed ID or missing status | Admin token; malformed path ID or omit `status` | `400` for malformed ID; `422` for validation failure |
| RENT-UPDATE-N3 | Update service fails | Admin token; valid ID/body; service returns error | `500`; `status:"failed"` and mapped error message |

### DELETE `/api/v1/rents/:id`

| ID | Case | Request | Expected |
|---|---|---|---|
| RENT-DELETE-P1 | Admin deletes rent | Admin token; valid ID; service succeeds | `200`; `status:"success"`, `message:"rent deleted"` |
| RENT-DELETE-N1 | Member is forbidden | Member token; valid ID | `403`; `message:"access denied"` |
| RENT-DELETE-N2 | Malformed ID | Admin token; `DELETE /api/v1/rents/nope` | `400`; `status:"failed"`, `message:"invalid id"` |
| RENT-DELETE-N3 | Delete service fails | Admin token; valid ID; service returns error | `500`; `status:"failed"` and mapped error message |

## Contract notes for test implementation

1. **Authentication failures:** missing credentials currently produce the same `401` / `Invalid or expired JWT` response as invalid/expired credentials at runtime, even though `jwtError` contains a `400` branch for the exact text `Missing or malformed JWT`. Tests should preserve the observed runtime behavior unless the API contract is intentionally changed.
2. **Not-found responses:** book/category/rent lookup errors are generally mapped to `500`, unlike profile lookup which maps to `404`. Tests should reflect current behavior; a consistent `404` policy would be a separate behavior change.
3. **Login rejection:** login service errors currently become `500`, including invalid credentials. Do not assert `401` without first changing and agreeing on that contract.
4. **Rent status validation:** the update request validates only that `status` is non-empty. The database enum includes `pending`, `on_delivery`, `rented`, `returned`, and `cancelled`, but the HTTP validator does not enforce that set. An unsupported non-empty status should be captured as a contract gap, not expected to return `422` yet.
5. **Pagination query parsing:** handlers ignore `strconv.Atoi` errors for `page` and `limit`; malformed values therefore become zero. Do not assume a `400` response unless input validation is added.
6. **Route-level tests:** verify middleware ordering and route precedence, especially `/books/category/:id` before `/books/:id`, using the actual route registration.
