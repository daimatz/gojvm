class TryCatch {
    static int safeDivide(int a, int b) {
        try {
            return a / b;
        } catch (ArithmeticException e) {
            return -1;
        }
    }
    static int nested() {
        int result = 0;
        try {
            try {
                int[] a = new int[3];
                int x = a[10];
            } catch (ArrayIndexOutOfBoundsException e) {
                result = 1;
            }
            result = result + safeDivide(10, 0);
        } catch (Exception e) {
            result = 99;
        }
        return result;
    }
    public static void main(String[] args) {
        System.out.println(safeDivide(10, 2));
        System.out.println(safeDivide(10, 0));
        System.out.println(nested());
    }
}
