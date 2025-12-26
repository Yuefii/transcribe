import os
import json
import time
import logging
from datetime import datetime
import redis
import mysql.connector
import whisper
import torch
import numpy as np
from sklearn.cluster import AgglomerativeClustering
from resemblyzer import VoiceEncoder, preprocess_wav

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

MODEL_SIZE = os.getenv('WHISPER_MODEL', 'small')
DIARIZATION_METHOD = os.getenv('DIARIZATION_METHOD', 'none')
HF_TOKEN = os.getenv('HUGGINGFACE_TOKEN', '')

class TranscriptionWorker:
    def __init__(self):
        self.redis_client = None
        self.db_connection = None
        self.model = None
        self.voice_encoder = None
        self.pyannote_pipeline = None
    
    def connect(self):
        try:
            self.redis_client = redis.Redis(
                 host=REDIS_HOST,
                port=REDIS_PORT,
                password=REDIS_PASSWORD if REDIS_PASSWORD else None,
                decode_responses=True
            )

            self.redis_client.ping()
            logger.info("connected to redis")

            self.db_connection = mysql.connector.connect(**DB_CONFIG)
            logger.info("connected to mysql")

            logger.info(f"loading whisper model: {MODEL_SIZE}")
            self.model = whisper.load_model(MODEL_SIZE)
            logger.info(f"whisper model loaded: {MODEL_SIZE}")

            if DIARIZATION_METHOD == 'resemblyzer':
                logger.info("loading resemblyzer voice encoder...")
                self.voice_encoder = VoiceEncoder()
                logger.info("resemblyzer loaded (no huggingface needed)")
            elif DIARIZATION_METHOD == 'pyannote':
                if not HF_TOKEN:
                    logger.warning("pyannote selected but no huggingface token provided")
                else:
                    from pyannote.audio import Pipeline
                    logger.info("loading pyannote Speaker diarization...")
                    self.pyannote_pipeline = Pipeline.from_pretrained(
                        "pyannote/speaker-diarization-3.1",
                        use_auth_token=HF_TOKEN
                    )
                    if torch.cuda.is_available():
                        self.pyannote_pipeline.to(torch.device("cuda"))
                        logger.info("pyannote loaded (GPU)")
                    else:
                        logger.info("pyannote loaded (CPU)")
            elif DIARIZATION_METHOD == 'simple':
                logger.info("simple speaker detection enabled (no extra dependencies)")
            else:
                logger.info("speaker diarization disabled")

        except Exception as e:
            logger.error(f"connection error: {e}")
            raise
    
    def update_job_status(self, job_id, status, text=None, error_msg=None, segments=None, duration=None):
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
                segments_json = json.dumps(segments) if segments else None
                
                query = """
                    UPDATE transcription_jobs 
                    SET status = %s, text = %s, segments = %s, duration = %s,
                        completed_at = %s, updated_at = %s 
                    WHERE id = %s
                """
                cursor.execute(query, (
                    status, text, segments_json, duration,
                    datetime.now(), datetime.now(), job_id
                ))
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

            self.publish_progress(job_id, status)

        except Exception as e:
            logger.error(f"database error updating job {job_id}: {e}")
            self.db_connection.rollback()
    
    def publish_progress(self, job_id, status, progress=None):
        try:
            message = {
                "job_id": job_id,
                "status": status,
                "timestamp": datetime.now().isoformat()
            }
            if progress is not None:
                message["progress"] = progress
            
            self.redis_client.publish(f"job_progress:{job_id}", json.dumps(message))
            logger.debug(f"published progress for {job_id}: {status}")
        except Exception as e:
            logger.error(f"redis publish error: {e}")
    
    def format_timestamp(self, seconds):
        minutes = int(seconds // 60)
        secs = int(seconds % 60)
        return f"{minutes:02d}:{secs:02d}"
    
    def simple_speaker_detection(self, audio_path, segments, job_id=None):
        try:
            import librosa
            
            logger.info("performing simple speaker detection...")
            self.publish_progress(segments[0].get('id', 0) if segments else "job", "detecting_speakers") # Hacky way to get job_id if not passed, but we should pass job_id
            
            audio, sr = librosa.load(audio_path, sr=16000)
            
            segment_features = []
            for seg in segments:
                start_sample = int(seg['start'] * sr)
                end_sample = int(seg['end'] * sr)
                segment_audio = audio[start_sample:end_sample]
                
                if len(segment_audio) > 0:
                    rms = librosa.feature.rms(y=segment_audio)[0]
                    zcr = librosa.feature.zero_crossing_rate(segment_audio)[0]
                    pitches = librosa.yin(segment_audio, fmin=50, fmax=400)
                    
                    features = [
                        np.mean(rms),
                        np.std(rms),
                        np.mean(zcr),
                        np.mean(pitches[~np.isnan(pitches)]) if len(pitches[~np.isnan(pitches)]) > 0 else 0
                    ]
                    segment_features.append(features)
                else:
                    segment_features.append([0, 0, 0, 0])
            
            n_speakers = min(4, len(segments))
            if n_speakers > 1:
                clustering = AgglomerativeClustering(n_clusters=n_speakers)
                labels = clustering.fit_predict(segment_features)
                
                for i, seg in enumerate(segments):
                    seg['speaker'] = f"SPEAKER_{labels[i]:02d}"
            else:
                for seg in segments:
                    seg['speaker'] = "SPEAKER_00"
            
            unique_speakers = len(set(seg['speaker'] for seg in segments))
            logger.info(f"simple detection found {unique_speakers} speakers")
            
            return segments
            
        except ImportError:
            logger.warning("librosa not installed, skipping simple speaker detection")
            return segments
        except Exception as e:
            logger.error(f"simple speaker detection error: {e}")
            return segments
        
    def resemblyzer_speaker_detection(self, audio_path, segments, job_id=None):
        try:
            logger.info("performing resemblyzer speaker detection...")
            if job_id: self.publish_progress(job_id, "detecting_speakers")
            
            wav = preprocess_wav(audio_path)
            
            embeddings = []
            for seg in segments:
                start_sample = int(seg['start'] * 16000)
                end_sample = int(seg['end'] * 16000)
                segment_wav = wav[start_sample:end_sample]
                
                if len(segment_wav) > 4000:
                    embedding = self.voice_encoder.embed_utterance(segment_wav)
                    embeddings.append(embedding)
                else:
                    if embeddings:
                        embeddings.append(embeddings[-1])
                    else:
                        embeddings.append(np.zeros(256))

            embeddings = np.array(embeddings)
            n_speakers = min(4, len(segments))

            if n_speakers > 1:
                clustering = AgglomerativeClustering(
                    n_clusters=n_speakers,
                    metric='cosine',
                    linkage='average'
                )
                labels = clustering.fit_predict(embeddings)
                
                for i, seg in enumerate(segments):
                    seg['speaker'] = f"SPEAKER_{labels[i]:02d}"
            else:
                for seg in segments:
                    seg['speaker'] = "SPEAKER_00"
            
            unique_speakers = len(set(seg['speaker'] for seg in segments))
            logger.info(f"resemblyzer detected {unique_speakers} speakers")
            
            return segments
        
        except Exception as e:
            logger.error(f"resemblyzer detection error: {e}")
            return segments
        
    def pyannote_speaker_detection(self, audio_path, segments, job_id=None):
        try:
            if not self.pyannote_pipeline:
                return segments
            
            logger.info("performing pyannote speaker diarization...")
            if job_id: self.publish_progress(job_id, "detecting_speakers")

            diarization = self.pyannote_pipeline(audio_path)

            for seg in segments:
                seg_mid = (seg['start'] + seg['end']) / 2
                
                for turn, _, speaker in diarization.itertracks(yield_label=True):
                    if turn.start <= seg_mid <= turn.end:
                        seg['speaker'] = speaker
                        break
                
                if 'speaker' not in seg:
                    seg['speaker'] = "SPEAKER_00"
            
            unique_speakers = len(set(seg.get('speaker', 'SPEAKER_00') for seg in segments))
            logger.info(f"pyannote detected {unique_speakers} speakers")
            
            return segments
        
        except Exception as e:
            logger.error(f"pyannote detection error: {e}")
            return segments

    def transcribe_audio(self, file_path, job_id):
        try:
            logger.info(f"transcribing: {file_path}")
            self.publish_progress(job_id, "loading_audio")
            
            self.publish_progress(job_id, "transcribing")
            result = self.model.transcribe(
                file_path,
                word_timestamps=True,
                verbose=False
            )
            
            full_text = result['text'].strip()
            duration = result.get('duration', 0)

            segments = []
            for i, segment in enumerate(result['segments']):
                seg = {
                    'id': i + 1,
                    'start': round(segment['start'], 2),
                    'end': round(segment['end'], 2),
                    'text': segment['text'].strip()
                }
                segments.append(seg)
            
            logger.info(f"transcription completed: {len(segments)} segments")
            logger.info(f"duration: {self.format_timestamp(duration)}")

            if DIARIZATION_METHOD == 'simple':
                segments = self.simple_speaker_detection(file_path, segments, job_id)
            elif DIARIZATION_METHOD == 'resemblyzer' and self.voice_encoder:
                segments = self.resemblyzer_speaker_detection(file_path, segments, job_id)
            elif DIARIZATION_METHOD == 'pyannote' and self.pyannote_pipeline:
                segments = self.pyannote_speaker_detection(file_path, segments, job_id)
            
            return full_text, segments, duration
            
        except Exception as e:
            logger.error(f"transcription error: {e}")
            raise

    def process_job(self, job_data):
        job_id = job_data['job_id']
        file_path = job_data['file_path']
        user_id = job_data['user_id']

        logger.info("=" * 60)
        logger.info(f"processing job: {job_id} (User: {user_id})")
        logger.info("=" * 60)
        
        try:
            if not os.path.exists(file_path):
                raise FileNotFoundError(f"file not found: {file_path}")
            
            self.update_job_status(job_id, 'processing')
            
            start_time = time.time()
            start_time = time.time()
            full_text, segments, duration = self.transcribe_audio(file_path, job_id)
            process_time = time.time() - start_time
            process_time = time.time() - start_time
            
            logger.info(f"processing completed in {process_time:.2f}s")
            logger.info(f"text length: {len(full_text)} characters")
            logger.info(f"segments: {len(segments)}")

            logger.info("\nsample segments:")
            for seg in segments[:3]:
                speaker_info = f" [{seg.get('speaker', 'N/A')}]" if 'speaker' in seg else ""
                logger.info(f"  [{self.format_timestamp(seg['start'])} - {self.format_timestamp(seg['end'])}]{speaker_info}: {seg['text'][:50]}...")

            self.publish_progress(job_id, "saving")
            self.update_job_status(
                job_id, 'done', 
                text=full_text, 
                segments=segments,
                duration=duration
            )
            
            logger.info(f"job {job_id} completed successfully")
            
        except Exception as e:
            error_msg = str(e)
            logger.error(f"job {job_id} failed: {error_msg}")
            self.update_job_status(job_id, 'failed', error_msg=error_msg)

    def run(self):
        logger.info("=" * 60)
        logger.info("transcription worker started")
        logger.info(f"whisper model: {MODEL_SIZE}")
        logger.info(f"diarization method: {DIARIZATION_METHOD}")
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