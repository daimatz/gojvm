class Switch {
    public static int rate(int score) {
        switch (score / 10) {
            case 10: case 9: return 5;
            case 8: return 4;
            case 7: return 3;
            case 6: return 2;
            default: return 1;
        }
    }
    public static void main(String[] args) {
        System.out.println(rate(95));
        System.out.println(rate(82));
        System.out.println(rate(45));
    }
}
