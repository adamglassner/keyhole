package com.ironite.keyhole;

import android.app.Activity;
import android.content.Intent;
import android.media.AudioManager;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;

import com.ironite.keyhole.mobile.Mobile;

import go.Seq;

public class MainActivity extends Activity {
    private final Handler handler = new Handler(Looper.getMainLooper());

    // The game asks to share a result by leaving text for us; open Android's share sheet.
    private final Runnable sharePoller = new Runnable() {
        @Override
        public void run() {
            String text = Mobile.takeShareText();
            if (text != null && !text.isEmpty()) {
                Intent send = new Intent(Intent.ACTION_SEND);
                send.setType("text/plain");
                send.putExtra(Intent.EXTRA_TEXT, text);
                startActivity(Intent.createChooser(send, getString(R.string.share_title)));
            }
            handler.postDelayed(this, 300);
        }
    };

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        Seq.setContext(getApplicationContext());
        Mobile.setDataDir(getFilesDir().getAbsolutePath());
        setVolumeControlStream(AudioManager.STREAM_MUSIC);
        setContentView(R.layout.activity_main);
    }

    private KeyholeView view() {
        return (KeyholeView) findViewById(R.id.ebitenview);
    }

    @Override
    protected void onPause() {
        super.onPause();
        handler.removeCallbacks(sharePoller);
        view().suspendGame();
    }

    @Override
    protected void onResume() {
        super.onResume();
        view().resumeGame();
        handler.post(sharePoller);
    }

    @Override
    @SuppressWarnings("deprecation")
    public void onBackPressed() {
        if (!Mobile.onBack()) {
            super.onBackPressed();
        }
    }
}
