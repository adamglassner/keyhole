package com.ironite.keyhole;

import android.content.Context;
import android.util.AttributeSet;
import android.util.Log;

import com.ironite.keyhole.mobile.EbitenView;

/** The Ebiten game view. Errors from the game are logged instead of crashing silently. */
public class KeyholeView extends EbitenView {
    public KeyholeView(Context context) {
        super(context);
    }

    public KeyholeView(Context context, AttributeSet attrs) {
        super(context, attrs);
    }

    @Override
    protected void onErrorOnGameUpdate(Exception e) {
        Log.e("Keyhole", "game error", e);
        super.onErrorOnGameUpdate(e);
    }
}
