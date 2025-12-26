# Audio Transcription API Specification

## Authentication
All protected endpoints require JWT token in Authorization header:
```
Authorization: Bearer <JWT_TOKEN>
```

**For WebSocket Connections:**
Since not all clients support custom headers during handshake, you can pass the token as a query parameter:
```
ws://localhost:8080/api/ws/job/:job_id?token=<JWT_TOKEN>
```

---

## Table of Contents
1. [Authentication](#authentication-endpoints)
2. [User Management](#user-management-endpoints)
3. [Transcription](#transcription-endpoints)
4. [Real-time Notifications](#real-time-notifications)
5. [Health Check](#health-check)
6. [Error Responses](#error-responses)
7. [Status Codes](#status-codes)

---

## Authentication Endpoints

### 1. Register User
Create a new user account.

**Endpoint:** `POST /auth/sign-up`

**Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "password123"
}
```

**Response:** `201 Created`
```json
{
  "message": "user registered successfully",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

**Validation Rules:**
- `name`: Required, string
- `email`: Required, valid email format, unique
- `password`: Required, minimum 6 characters

**Error Response:** `400 Bad Request`
```json
{
  "error": "email already exists"
}
```

---

### 2. Login User
Authenticate user and get access token.

**Endpoint:** `POST /auth/sign-in`

**Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "email": "john@example.com",
  "password": "password123"
}
```

**Response:** `200 OK`
```json
{
  "message": "signin successfully",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

**Error Response:** `401 Unauthorized`
```json
{
  "error": "invalid email or password"
}
```

---

## User Management Endpoints

### 3. Get User Profile
Retrieve authenticated user's profile.

**Endpoint:** `GET /user/profile`

**Headers:**
```
Authorization: Bearer <JWT_TOKEN>
```

**Response:** `200 OK`
```json
{
  "user": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

**Error Response:** `401 Unauthorized`
```json
{
  "error": "invalid or expired token"
}
```

---

### 4. Update User Profile
Update authenticated user's profile information.

**Endpoint:** `PUT /user/profile`

**Headers:**
```
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "John Updated"
}
```

**Response:** `200 OK`
```json
{
  "message": "profile update succesfully"
}
```

---

---

## Transcription Endpoints

### 5. Create Transcription Job
Upload audio/video file and create transcription job.

**Endpoint:** `POST /transcribe`

**Headers:**
```
Authorization: Bearer <JWT_TOKEN>
Content-Type: multipart/form-data
```

**Request Body:**
- `audio` (file): Audio or video file

**Supported Formats:**
- Audio: `.mp3`, `.wav`, `.m4a`, `.ogg`, `.flac`
- Video: `.mp4`, `.avi`, `.mov`

**File Size Limit:** 100 MB

**Response:** `201 Created`
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "message": "job created and queued for transcription"
}
```

**Error Responses:**

`400 Bad Request` - Invalid file type:
```json
{
  "error": "Invalid file type. Allowed: mp3, wav, m4a, ogg, flac, mp4, avi, mov"
}
```

`400 Bad Request` - File too large:
```json
{
  "error": "file size exceeds 100MB limit"
}
```

`400 Bad Request` - No file uploaded:
```json
{
  "error": "audio file is required"
}
```

**cURL Example:**
```bash
curl -X POST http://localhost:8080/api/transcribe \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "audio=@/path/to/audio.mp3"
```

---

### 6. Get Job Status
Retrieve transcription job status and result.

**Endpoint:** `GET /transcribe/{job_id}`

**Headers:**
```
Authorization: Bearer <JWT_TOKEN>
```

**Path Parameters:**
- `job_id`: UUID of the transcription job

**Example:**
```
GET /transcribe/550e8400-e29b-41d4-a716-446655440000
```

**Response (Queued):** `200 OK`
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "file_name": "audio.mp3",
  "file_size": 2048576,
  "created_at": "2025-12-23T10:00:00Z"
}
```

**Response (Processing):** `200 OK`
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "file_name": "audio.mp3",
  "file_size": 2048576,
  "created_at": "2025-12-23T10:00:00Z"
}
```

**Response (Done):** `200 OK`
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "done",
  "text": "Halo semua, ini adalah hasil transkripsi dari audio yang telah diupload. Sistem ini menggunakan teknologi speech-to-text yang canggih.",
  "file_name": "audio.mp3",
  "file_size": 2048576,
  "created_at": "2025-12-23T10:00:00Z",
  "completed_at": "2025-12-23T10:02:30Z"
}
```

**Response (Failed):** `200 OK`
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "failed",
  "error_message": "File not found or corrupted",
  "file_name": "audio.mp3",
  "file_size": 2048576,
  "created_at": "2025-12-23T10:00:00Z",
  "completed_at": "2025-12-23T10:02:30Z"
}
```

**Status Values:**
- `queued`: Job created and waiting in queue
- `processing`: Worker has picked up the job (general status)
- `loading_audio`: Worker is downloading/converting audio file
- `transcribing`: AI model is actively listening and transcribing
- `detecting_speakers`: AI is analyzing different speakers (if enabled)
- `saving`: Saving results to database
- `done`: Transcription completed successfully
- `failed`: Error occurred during transcription
- `cancelled`: Job was cancelled by user


**Error Response:** `404 Not Found`
```json
{
  "error": "job not found"
}
```

**Error Response:** `403 Forbidden`
```json
{
  "error": "access denied"
}
```

---

### 7. Get User Jobs
Retrieve all transcription jobs for authenticated user.

**Endpoint:** `GET /transcribe`

**Headers:**
```
Authorization: Bearer <JWT_TOKEN>
```

**Query Parameters:**
- `page` (optional): Page number, default: 1
- `page_size` (optional): Items per page, default: 10, max: 100

**Example:**
```
GET /transcribe?page=1&page_size=10
```

**Response:** `200 OK`
```json
{
  "jobs": [
    {
      "job_id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "done",
      "text": "Halo semua, ini hasil transkripsi...",
      "file_name": "audio1.mp3",
      "file_size": 2048576,
      "created_at": "2025-12-23T10:00:00Z",
      "completed_at": "2025-12-23T10:02:30Z"
    },
    {
      "job_id": "660e8400-e29b-41d4-a716-446655440001",
      "status": "processing",
      "file_name": "audio2.mp3",
      "file_size": 3145728,
      "created_at": "2025-12-23T11:00:00Z"
    },
    {
      "job_id": "770e8400-e29b-41d4-a716-446655440002",
      "status": "queued",
      "file_name": "audio3.mp3",
      "file_size": 1572864,
      "created_at": "2025-12-23T11:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total": 3,
    "total_page": 1
  }
}
```

---

### 8. Delete Transcription Job
Delete a transcription job and its associated file.

**Endpoint:** `DELETE /transcribe/{job_id}`

**Headers:**
```
Authorization: Bearer <JWT_TOKEN>
```

**Path Parameters:**
- `job_id`: UUID of the transcription job

**Example:**
```
DELETE /transcribe/550e8400-e29b-41d4-a716-446655440000
```

**Response:** `200 OK`
```json
{
  "message": "job deleted successfully"
}
```

**Error Response:** `404 Not Found`
```json
{
  "error": "job not found"
}
```

**Error Response:** `403 Forbidden`
```json
{
  "error": "access denied"
}
```

---

### 9. Cancel Transcription Job
Stop a generic processing job immediately.

**Endpoint:** `POST /transcribe/{job_id}/cancel`

**Headers:**
```
Authorization: Bearer <JWT_TOKEN>
```

**Response:** `200 OK`
```json
{
  "message": "job cancellation request sent",
  "status": "cancelled"
}
```

**Error Response:** `400 Bad Request`
```json
{
  "error": "job is already finished or cancelled"
}
```

---

## Real-time Notifications

### 10. WebSocket Progress Stream
Connect to receive real-time granular updates about job status.

**Endpoint:** `WS /ws/job/:job_id`

**Query Parameters:**
- `token`: JWT Token (Required)

**Messages:**

**1. Initial Message (On Connect):**
```json
{
  "job_id": "uuid...",
  "status": "transcribing",
  "message": "connected. current status fetched."
}
```

**2. Progress Update:**
```json
{
  "job_id": "uuid...",
  "status": "transcribing",
  "timestamp": "2025-12-23T10:00:05Z"
}
```

**3. Granular Statuses:**
You will receive these statuses in order:
`processing` -> `loading_audio` -> `transcribing` -> `detecting_speakers` -> `saving` -> `done`

---

## Health Check

### 11. Health Check
Check if API server is running.

**Endpoint:** `GET /health`

**Headers:** None required

**Response:** `200 OK`
```json
{
  "status": "ok",
  "message": "server is running"
}
```

---

## Error Responses

### Standard Error Format
All error responses follow this format:
```json
{
  "error": "Error message description"
}
```

### Common Error Messages

| Status Code | Error Message | Description |
|-------------|---------------|-------------|
| 400 | Invalid request body | Malformed JSON or missing required fields |
| 401 | Missing authorization header | No JWT token provided |
| 401 | Invalid or expired token | Token is invalid or has expired |
| 403 | Access denied | User doesn't have permission to access resource |
| 404 | Job not found | Requested job ID doesn't exist |
| 404 | User not found | Requested user doesn't exist |
| 500 | Internal server error | Unexpected server error |

---

## Status Codes

| Code | Status | Description |
|------|--------|-------------|
| 200 | OK | Request successful |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Invalid request parameters |
| 401 | Unauthorized | Authentication required or failed |
| 403 | Forbidden | Access denied |
| 404 | Not Found | Resource not found |
| 500 | Internal Server Error | Server error |

---

## Notes

### File Upload
- Maximum file size: 100 MB
- Allowed formats: mp3, wav, m4a, ogg, flac, mp4, avi, mov
- Files are stored in `./uploads/user_{user_id}/` directory
- File naming: `{job_id}.{extension}`

### JWT Token
- Token expires after 24 hours
- Include in Authorization header as: `Bearer <token>`
- Token contains: user_id, email, expiration time

### Transcription Processing
- Jobs are processed asynchronously by Python worker
- Processing time depends on audio length and model size
- Average: ~2-5 minutes for 10-minute audio (base model)

---

**Version:** 1.0.0  
**Last Updated:** December 2025