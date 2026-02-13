export class APIClient {
    private baseUrl: string;
    private sessionToken: string | null = null;

    constructor() {
        if (typeof window !== 'undefined') {
            const host = window.location.hostname;
            const port = window.location.port === '5173' ? '3000' : window.location.port;
            this.baseUrl = `${window.location.protocol}//${host}${port ? ':' + port : ''}/api`;
            this.sessionToken = localStorage.getItem('session_token');
        } else {
            this.baseUrl = 'http://localhost:3000/api';
        }
    }

    getBaseUrl() {
        return this.baseUrl;
    }

    getAuthenticatedMediaUrl(path: string) {
        const separator = path.includes('?') ? '&' : '?';
        return `${this.baseUrl}${path}${separator}session_token=${this.sessionToken}`;
    }

    setToken(token: string | null) {
        this.sessionToken = token;
        if (typeof window !== 'undefined') {
            if (token) localStorage.setItem('session_token', token);
            else localStorage.removeItem('session_token');
        }
    }

    public async request(method: string, path: string, body?: any) {
        const headers: HeadersInit = {
            'X-Requested-With': 'XMLHttpRequest'
        };

        if (this.sessionToken) {
            headers['Authorization'] = `Bearer ${this.sessionToken}`;
        }

        if (body && !(body instanceof FormData)) {
            headers['Content-Type'] = 'application/json';
        }

        const options: RequestInit = {
            method,
            headers,
            body: body instanceof FormData ? body : (body ? JSON.stringify(body) : undefined)
        };

        const response = await fetch(`${this.baseUrl}${path}`, options);
        const data = await response.json();

        if (!response.ok) {
            if (response.status === 401) {
                this.setToken(null);
                if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login') && !window.location.pathname.startsWith('/setup')) {
                    window.location.href = '/login';
                }
                throw new Error('Your session has expired. Please log in again.');
            }
            throw new Error(data.error?.message || 'Request failed');
        }

        return data.data;
    }

    // Auth
    async login(payload: any) {
        const data = await this.request('POST', '/auth/login', payload);
        this.setToken(data.token);
        return data;
    }

    async setup(payload: any) {
        return this.request('POST', '/auth/setup', payload);
    }

    async getStatus() {
        return this.request('GET', '/auth/status');
    }

    async logout() {
        await this.request('POST', '/auth/logout');
        this.setToken(null);
    }

    // Exams
    async listExams() { return this.request('GET', '/exams'); }
    async createExam(payload: any) { return this.request('POST', '/exams', payload); }
    async getExam(id: string) { return this.request('GET', `/exams/details?exam_id=${id}`); }
    async updateExam(payload: any) { return this.request('PATCH', '/exams', payload); }
    async deleteExam(id: string) { return this.request('DELETE', '/exams', { exam_id: id }); }
    async searchExam(id: string, query: string) { return this.request('GET', `/exams/search?exam_id=${id}&query=${encodeURIComponent(query)}`); }

    // Lectures
    async listLectures(examId: string) { return this.request('GET', `/lectures?exam_id=${examId}`); }
    async createLecture(formData: FormData) { return this.request('POST', '/lectures', formData); }
    async getLecture(lectureId: string, examId: string) { return this.request('GET', `/lectures/details?lecture_id=${lectureId}&exam_id=${examId}`); }
    async deleteLecture(lectureId: string, examId: string) { return this.request('DELETE', '/lectures', { lecture_id: lectureId, exam_id: examId }); }
    async retryLectureJob(lectureId: string, examId: string, jobType: string) { return this.request('POST', '/lectures/retry-job', { lecture_id: lectureId, exam_id: examId, job_type: jobType }); }

    // Transcripts
    async getTranscript(lectureId: string) { return this.request('GET', `/transcripts?lecture_id=${lectureId}`); }
    async getTranscriptHTML(lectureId: string) { return this.request('GET', `/transcripts/html?lecture_id=${lectureId}`); }

    // Documents
    async listDocuments(lectureId: string) { return this.request('GET', `/documents?lecture_id=${lectureId}`); }
    async getDocumentPages(docId: string, lectureId: string) { return this.request('GET', `/documents/pages?document_id=${docId}&lecture_id=${lectureId}`); }

    // Study Tools
    async listTools(examId: string, type?: string) { return this.request('GET', `/tools?exam_id=${examId}${type ? `&type=${type}` : ''}`); }
    async createTool(payload: any) { return this.request('POST', '/tools', payload); }
    async getToolHTML(toolId: string, examId: string) { return this.request('GET', `/tools/html?tool_id=${toolId}&exam_id=${examId}`); }
    async exportTool(payload: any) { return this.request('POST', '/tools/export', payload); }

    // Jobs
    async listJobs() { return this.request('GET', '/jobs'); }
    async getJob(id: string) { return this.request('GET', `/jobs/details?job_id=${id}`); }

    // Settings
    async getSettings() { return this.request('GET', '/settings'); }
    async updateSettings(payload: any) { return this.request('PATCH', '/settings', payload); }
}

export const api = new APIClient();
