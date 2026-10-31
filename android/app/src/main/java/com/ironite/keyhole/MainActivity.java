package com.ironite.keyhole;

import android.app.Activity;
import android.content.Intent;
import android.media.AudioManager;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.provider.Settings;
import android.os.Handler;
import android.os.Looper;

import com.ironite.keyhole.mobile.Mobile;

import java.io.File;

import go.Seq;

public class MainActivity extends Activity {
    private final Handler handler = new Handler(Looper.getMainLooper());

    // An update APK waiting for the player to allow installing apps.
    private String pendingInstall;
    private boolean askedPermission;

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
            String apk = Mobile.takeInstallPath();
            if (apk != null && !apk.isEmpty()) {
                pendingInstall = apk;
                askedPermission = false;
                tryInstall();
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

    private boolean canInstall() {
        return Build.VERSION.SDK_INT < Build.VERSION_CODES.O || getPackageManager().canRequestPackageInstalls();
    }

    // Opens Android's install prompt for a downloaded update. On Android 8+ the
    // player first has to allow Keyhole to install apps; we send them to that
    // setting once and try again when they come back.
    private void tryInstall() {
        if (pendingInstall == null) {
            return;
        }
        if (!canInstall()) {
            if (!askedPermission) {
                askedPermission = true;
                Intent s = new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:" + getPackageName()));
                try {
                    startActivity(s);
                    return;
                } catch (Exception e) {
                    // fall through
                }
            }
            pendingInstall = null;
            Mobile.installBlocked();
            return;
        }
        File f = new File(pendingInstall);
        pendingInstall = null;
        Intent i = new Intent(Intent.ACTION_VIEW);
        i.setDataAndType(UpdateProvider.uriFor(f), "application/vnd.android.package-archive");
        i.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
        try {
            startActivity(i);
        } catch (Exception e) {
            Mobile.installBlocked();
        }
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
        if (pendingInstall != null && askedPermission) {
            tryInstall(); // back from the "install unknown apps" setting
        }
    }

    @Override
    @SuppressWarnings("deprecation")
    public void onBackPressed() {
        if (!Mobile.onBack()) {
            super.onBackPressed();
        }
    }
}
