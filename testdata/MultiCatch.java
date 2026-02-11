public class MultiCatch {
    // Labeled break/continue
    static void labeledBreak() {
        int sum = 0;
        outer:
        for (int i = 0; i < 5; i++) {
            for (int j = 0; j < 5; j++) {
                if (i + j > 3) break outer;
                sum += 1;
            }
        }
        System.out.println(sum); // 4 (i=0:j=0,1,2,3 then break outer at j=4)
    }

    // do-while
    static void doWhile() {
        int n = 1;
        int sum = 0;
        do {
            sum += n;
            n++;
        } while (n <= 5);
        System.out.println(sum); // 15
    }

    // Multi-catch (Java 7)
    static void multiCatch(int n) {
        try {
            if (n == 0) {
                throw new IllegalArgumentException("bad arg");
            } else if (n == 1) {
                throw new UnsupportedOperationException("unsupported");
            } else {
                int x = 10 / n;
                System.out.println(x);
            }
        } catch (IllegalArgumentException | UnsupportedOperationException e) {
            System.out.println("caught:" + e.getMessage());
        }
    }

    // Ternary operator
    static String classify(int n) {
        return n > 0 ? "positive" : (n < 0 ? "negative" : "zero");
    }

    public static void main(String[] args) {
        labeledBreak();   // 10
        doWhile();        // 15

        multiCatch(0);    // caught:bad arg
        multiCatch(1);    // caught:unsupported
        multiCatch(5);    // 2

        System.out.println(classify(42));   // positive
        System.out.println(classify(-1));   // negative
        System.out.println(classify(0));    // zero
    }
}
