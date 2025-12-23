import os
import json
import time
import logging
from datetime import datetime
import redis
import mysql.connector
from mysql.connector import Error
import whisper

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

REDIS_HOST = os.getenv('REDIS_HOST', 'localhost')
REDIS_PORT = int(os.getenv('REDIS_PORT', 6379))
REDIS_PASSWORD = os.getenv('REDIS_PASSWORD', '')
REDIS_QUEUE = 'transcription_queue'

DB_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': int(os.getenv('DB_PORT', 3306)),
    'user': os.getenv('DB_USER', 'root'),
    'password': os.getenv('DB_PASSWORD', '@dev2323'),
    'database': os.getenv('DB_NAME', 'transcribe')
}

MODEL_SIZE = os.getenv('WHISPER_MODEL', 'base')

class TranscriptionWorker:
    def __init__(self):
        self.redis_client = None
        self.db_connection = None
        self.model = None
    
    def connect(self):
        try:
            self.redis_client = redis.Redis(
                 host=REDIS_HOST,
                port=REDIS_PORT,
                password=REDIS_PASSWORD if REDIS_PASSWORD else None,
                decode_responses=True
            )

            self.db_connection = mysql.connector.connect(**DB_CONFIG)
            logger.info("connected to mysql")

            logger.info(f"loading whisper model: {MODEL_SIZE}")
            self.model = whisper.load_model(MODEL_SIZE)
            logger.info(f"whisper model loaded: {MODEL_SIZE}")

        except Exception as e:
            logger.error(f"connection error: {e}")
            raise
    
    def update_job_status(self, job_id, status, text=None, error_msg=None):
        try:
            cursor = self.db_connection.cursor()

            if status == 'processing':
                query = """
                    UPDATE transcription_jobs
                    SET status = %s, updated_at = %s
                    WHERE id = %s
                """
                cursor.execute(query, (status, datetime.now(), job_id))
            elif status == 'done':
                query = """
                    UPDATE transcription_jobs 
                    SET status = %s, text = %s, completed_at = %s, updated_at = %s 
                    WHERE id = %s
                """
                cursor.execute(query, (status, text, datetime.now(), datetime.now(), job_id))
            elif status == 'failed':
                query = """
                    UPDATE transcription_jobs 
                    SET status = %s, error_msg = %s, completed_at = %s, updated_at = %s 
                    WHERE id = %s
                """
                cursor.execute(query, (status, error_msg, datetime.now(), datetime.now(), job_id))
            
            self.db_connection.commit()
            cursor.close()
            logger.info(f"job {job_id} status updated to: {status}")

        except Exception as e:
            logger.error(f"database error updating job {job_id}: {e}")
            self.db_connection.rollback()

    def transcribe_audio(self, file_path):
        try:
            logger.info(f"Transcribing: {file_path}")
            
            result = self.model.transcribe(file_path)
            
            return result['text'].strip()
            
        except Exception as e:
            logger.error(f"Transcription error: {e}")
            raise

    def process_job(self, job_data):
        job_id = job_data['job_id']
        file_path = job_data['file_path']
        user_id = job_data['user_id']

        logger.info(f"processing job: {job_id} (user: {user_id})")
        
        try:
            if not os.path.exists(file_path):
                raise FileNotFoundError(f"file not found: {file_path}")
            
            self.update_job_status(job_id, 'processing')
            
            start_time = time.time()
            transcription_text = self.transcribe_audio(file_path)
            duration = time.time() - start_time
            
            logger.info(f"transcription completed in {duration:.2f}s")
            logger.info(f"text length: {len(transcription_text)} characters")
            
            self.update_job_status(job_id, 'done', text=transcription_text)
            
            logger.info(f"job {job_id} completed successfully")
            
        except Exception as e:
            error_msg = str(e)
            logger.error(f"job {job_id} failed: {error_msg}")
            self.update_job_status(job_id, 'failed', error_msg=error_msg)

    def run(self):
        logger.info("=" * 60)
        logger.info("transcription worker started")
        logger.info(f"model: {MODEL_SIZE}")
        logger.info(f"queue: {REDIS_QUEUE}")
        logger.info("=" * 60)
        
        while True:
            try:
                job = self.redis_client.blpop(REDIS_QUEUE, timeout=5)
                
                if job:
                    job_data = json.loads(job[1])
                    self.process_job(job_data)
                    
            except KeyboardInterrupt:
                logger.info("\nshutting down worker...")
                break
                
            except Exception as e:
                logger.error(f"worker error: {e}")
                time.sleep(5)
    
        if self.db_connection and self.db_connection.is_connected():
            self.db_connection.close()
            logger.info("database connection closed")
        
        logger.info("worker stopped")

if __name__ == '__main__':
    worker = TranscriptionWorker()
    worker.connect()
    worker.run()